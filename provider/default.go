package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/urltest"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	O "github.com/sagernet/sing-box/outbound"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/atomic"
	"github.com/sagernet/sing/common/batch"
	E "github.com/sagernet/sing/common/exceptions"
	F "github.com/sagernet/sing/common/format"
	"github.com/sagernet/sing/common/rw"
	"github.com/sagernet/sing/service/pause"
)

type SubscriptionInfo struct {
	upload   uint64
	download uint64
	total    uint64
	expire   uint64
}

type myProviderAdapter struct {
	ctx                 context.Context
	router              adapter.Router
	logger              log.ContextLogger
	subscriptionInfo    SubscriptionInfo
	tag                 string
	path                string
	healthcheckUrl      string
	healthcheckInterval time.Duration
	overrideDialer      map[string]any
	healchcheckHistory  *urltest.HistoryStorage
	providerType        string
	updateTime          time.Time
	outbounds           []adapter.Outbound
	outboundByTag       map[string]adapter.Outbound
	checking            atomic.Bool
	updating            atomic.Bool
	pauseManager        pause.Manager
	lastHealthcheck     time.Time

	sync.RWMutex
	access sync.Mutex
	ticker *time.Ticker
	close  chan struct{}
}

func (a *myProviderAdapter) Tag() string {
	return a.tag
}

func (a *myProviderAdapter) Path() string {
	return a.path
}

func (a *myProviderAdapter) Type() string {
	return a.providerType
}

func (a *myProviderAdapter) UpdateTime() time.Time {
	a.RLock()
	defer a.RUnlock()
	return a.updateTime
}

func (a *myProviderAdapter) Outbound(tag string) (adapter.Outbound, bool) {
	a.RLock()
	defer a.RUnlock()
	outbound, loaded := a.outboundByTag[tag]
	return outbound, loaded
}

func (a *myProviderAdapter) Outbounds() []adapter.Outbound {
	a.RLock()
	defer a.RUnlock()
	outbounds := []adapter.Outbound{}
	outbounds = append(outbounds, a.outbounds...)
	return outbounds
}

func GetFirstLine(content string) (string, string) {
	lines := strings.Split(content, "\n")
	if len(lines) == 1 {
		return lines[0], ""
	}
	others := strings.Join(lines[1:], "\n")
	return lines[0], others
}

func (a *myProviderAdapter) SubscriptionInfo() map[string]uint64 {
	info := make(map[string]uint64)
	info["Upload"] = a.subscriptionInfo.upload
	info["Download"] = a.subscriptionInfo.download
	info["Total"] = a.subscriptionInfo.total
	info["Expire"] = a.subscriptionInfo.expire
	return info
}

func (a *myProviderAdapter) ParseSubInfo(infoString string) bool {
	reg := regexp.MustCompile("upload=(\\d*);[ \t]*download=(\\d*);[ \t]*total=(\\d*);[ \t]*expire=(\\d*)")
	result := reg.FindStringSubmatch(infoString)
	if len(result) > 0 {
		upload, _ := strconv.Atoi(result[1:][0])
		download, _ := strconv.Atoi(result[1:][1])
		total, _ := strconv.Atoi(result[1:][2])
		expire, _ := strconv.Atoi(result[1:][3])
		a.subscriptionInfo.upload = uint64(upload)
		a.subscriptionInfo.download = uint64(download)
		a.subscriptionInfo.total = uint64(total)
		a.subscriptionInfo.expire = uint64(expire)
		return true
	}
	return false
}

func (a *myProviderAdapter) CreateOutboundFromContent(ctx context.Context, router adapter.Router, outbounds []option.Outbound) error {
	for _, outbound := range outbounds {
		otype := outbound.Type
		tag := outbound.Tag
		switch otype {
		case C.TypeDirect, C.TypeBlock, C.TypeDNS, C.TypeSelector, C.TypeURLTest:
			continue
		default:
			out, err := O.New(ctx, router, a.logger, tag, outbound)
			if err != nil {
				E.New("invalid outbound")
				continue
			}
			a.outboundByTag[tag] = out
			a.outbounds = append(a.outbounds, out)
		}
	}
	return nil
}

func GetTrimedFile(path string) []byte {
	content, _ := os.ReadFile(path)
	return []byte(TrimBlank(string(content)))
}

func TrimBlank(str string) string {
	str = strings.Trim(str, " ")
	str = strings.Trim(str, "\a")
	str = strings.Trim(str, "\b")
	str = strings.Trim(str, "\f")
	str = strings.Trim(str, "\r")
	str = strings.Trim(str, "\t")
	str = strings.Trim(str, "\v")
	return str
}

func (p *myProviderAdapter) GetContentFromFile(router adapter.Router) string {
	p.updateTime = time.Unix(int64(0), int64(0))
	path := p.path
	if !rw.FileExists(path) {
		return ""
	}
	fileInfo, _ := os.Stat(path)
	p.updateTime = fileInfo.ModTime()
	contentRaw := GetTrimedFile(path)
	content := string(contentRaw)
	firstLine, others := GetFirstLine(content)
	if p.ParseSubInfo(firstLine) {
		content = others
	}
	return content
}

func replaceIllegalBase64(content string) string {
	result := content
	result = strings.ReplaceAll(result, "-", "+")
	result = strings.ReplaceAll(result, "_", "/")
	return result
}

func DecodeBase64Safe(content string) string {
	reg1 := regexp.MustCompile(`^(?:[A-Za-z0-9-_+/]{4})*[A-Za-z0-9_+/]{4}$`)
	reg2 := regexp.MustCompile(`^(?:[A-Za-z0-9-_+/]{4})*[A-Za-z0-9_+/]{3}(=)?$`)
	reg3 := regexp.MustCompile(`^(?:[A-Za-z0-9-_+/]{4})*[A-Za-z0-9_+/]{2}(==)?$`)
	var result []string
	result = reg1.FindStringSubmatch(content)
	if len(result) > 0 {
		decode, err := base64.StdEncoding.DecodeString(replaceIllegalBase64(content))
		if err == nil {
			return string(decode)
		}
	}
	result = reg2.FindStringSubmatch(content)
	if len(result) > 0 {
		equals := ""
		if result[1] == "" {
			equals = "="
		}
		decode, err := base64.StdEncoding.DecodeString(replaceIllegalBase64(content + equals))
		if err == nil {
			return string(decode)
		}
	}
	result = reg3.FindStringSubmatch(content)
	if len(result) > 0 {
		equals := ""
		if result[1] == "" {
			equals = "=="
		}
		decode, err := base64.StdEncoding.DecodeString(replaceIllegalBase64(content + equals))
		if err == nil {
			return string(decode)
		}
	}
	return content
}

func ParseContent(contentRaw string) string {
	content := DecodeBase64Safe(contentRaw)
	return content
}

func (p *myProviderAdapter) ParseOutbounds(ctx context.Context, router adapter.Router, content string) error {
	if len(content) == 0 {
		return nil
	}
	outbounds, err := newParser(content, p.overrideDialer)
	if err != nil {
		return err
	}
	err = p.CreateOutboundFromContent(ctx, router, outbounds)
	if err != nil {
		return err
	}
	return nil
}

func (p *myProviderAdapter) BackupProvider() ([]adapter.Outbound, map[string]adapter.Outbound, SubscriptionInfo) {
	outboundsBackup := []adapter.Outbound{}
	outboundByTagBackup := make(map[string]adapter.Outbound)
	outboundsBackup = append(outboundsBackup, p.outbounds...)
	for tag, out := range p.outboundByTag {
		outboundByTagBackup[tag] = out
	}
	subscriptionInfoBackup := SubscriptionInfo{
		upload:   p.subscriptionInfo.upload,
		download: p.subscriptionInfo.download,
		total:    p.subscriptionInfo.total,
		expire:   p.subscriptionInfo.expire,
	}
	p.outbounds = []adapter.Outbound{}
	p.outboundByTag = make(map[string]adapter.Outbound)
	p.subscriptionInfo.upload = uint64(0)
	p.subscriptionInfo.download = uint64(0)
	p.subscriptionInfo.total = uint64(0)
	p.subscriptionInfo.expire = uint64(0)
	return outboundsBackup, outboundByTagBackup, subscriptionInfoBackup
}

func (p *myProviderAdapter) RevertProvider(outboundsBackup []adapter.Outbound, outboundByTagBackup map[string]adapter.Outbound, subscriptionInfoBackup SubscriptionInfo) {
	for _, out := range p.outbounds {
		common.Close(out)
	}
	p.outbounds = outboundsBackup
	p.outboundByTag = outboundByTagBackup
	p.subscriptionInfo = subscriptionInfoBackup
}

func (p *myProviderAdapter) UpdateOutboundByTag() {
	p.outboundByTag = make(map[string]adapter.Outbound)
	for _, out := range p.outbounds {
		tag := out.Tag()
		p.outboundByTag[tag] = out
	}
}

func (p *myProviderAdapter) StartOutbounds(router adapter.Router) error {
	pTag := p.Tag()
	outboundTag := make(map[string]bool)
	for _, out := range router.Outbounds() {
		outboundTag[out.Tag()] = true
	}
	for _, p := range router.OutboundProviders() {
		if p.Tag() == pTag {
			continue
		}
		for _, out := range p.Outbounds() {
			outboundTag[out.Tag()] = true
		}
	}
	for i, out := range p.Outbounds() {
		var tag string
		if out.Tag() == "" {
			tag = fmt.Sprint("[", pTag, "]", F.ToString(i))
		} else {
			tag = out.Tag()
		}
		if _, exists := outboundTag[tag]; exists {
			i := 1
			for {
				tTag := fmt.Sprint(tag, "[", i, "]")
				if _, exists := outboundTag[tTag]; exists {
					i++
					continue
				}
				tag = tTag
				break
			}
			out.SetTag(tag)
		}
		outboundTag[tag] = true
		if starter, isStarter := out.(common.Starter); isStarter {
			p.logger.Trace("initializing outbound provider[", pTag, "]", " outbound/", out.Type(), "[", tag, "]")
			err := starter.Start()
			if err != nil {
				return E.Cause(err, "initialize outbound provider[", pTag, "]", " outbound/", out.Type(), "[", tag, "]")
			}
		}
	}
	p.UpdateOutboundByTag()
	return nil
}

func (p *myProviderAdapter) UpdateGroups(router adapter.Router) error {
	for _, outbound := range router.Outbounds() {
		if group, ok := outbound.(adapter.OutboundGroup); ok {
			err := group.UpdateOutbounds(p.tag)
			if err != nil {
				return E.Cause(err, "update provider ", p.tag, " failed")
			}
		}
	}
	return nil
}

func (p *myProviderAdapter) PRLock() {
	p.RLock()
}

func (p *myProviderAdapter) PRUnlock() {
	p.RUnlock()
}

func (p *myProviderAdapter) RunFuncsWithRevert(funcArray ...func() error) error {
	for _, funcToRun := range funcArray {
		err := funcToRun()
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *myProviderAdapter) CheckOutbounds(force bool) {
	_, _ = p.healthcheck(p.ctx, p.healthcheckUrl, force)
}

func (p *myProviderAdapter) Healthcheck(ctx context.Context, link string, force bool) (map[string]uint16, error) {
	url := p.healthcheckUrl
	if link != "" {
		url = link
	}
	return p.healthcheck(ctx, url, force)
}

func (p *myProviderAdapter) healthcheck(ctx context.Context, link string, force bool) (map[string]uint16, error) {
	result := make(map[string]uint16)
	if p.checking.Swap(true) {
		return result, nil
	}
	defer p.checking.Store(false)

	if !force && time.Since(p.lastHealthcheck) < p.healthcheckInterval {
		return result, nil
	}
	p.lastHealthcheck = time.Now()
	b, _ := batch.New(ctx, batch.WithConcurrencyNum[any](10))
	checked := make(map[string]bool)
	var resultAccess sync.Mutex
	p.RLock()
	defer p.RUnlock()
	for _, detour := range p.outbounds {
		tag := detour.Tag()
		if checked[tag] {
			continue
		}
		checked[tag] = true
		detour, loaded := p.outboundByTag[tag]
		if !loaded {
			continue
		}
		b.Go(tag, func() (any, error) {
			ctx, cancel := context.WithTimeout(context.Background(), C.TCPTimeout)
			defer cancel()
			t, err := urltest.URLTest(ctx, link, detour)
			if err != nil {
				p.logger.Debug("outbound ", tag, " unavailable: ", err)
				p.healchcheckHistory.DeleteURLTestHistory(tag)
			} else {
				p.logger.Debug("outbound ", tag, " available: ", t, "ms")
				p.healchcheckHistory.StoreURLTestHistory(tag, &urltest.History{
					Time:  time.Now(),
					Delay: t,
				})
				resultAccess.Lock()
				result[tag] = t
				resultAccess.Unlock()
			}
			return nil, nil
		})
	}
	b.Wait()
	return result, nil
}
