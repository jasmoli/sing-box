package provider

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/urltest"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/service"
	"github.com/sagernet/sing/service/pause"
)

var _ adapter.OutboundProvider = (*HTTPProvider)(nil)

type HTTPProvider struct {
	myProviderAdapter
	url          string
	ua           string
	interval     time.Duration
	lastDownload time.Time
	detour       string
	start        bool
}

func (p *HTTPProvider) Start() {
	var history *urltest.HistoryStorage
	if history = service.PtrFromContext[urltest.HistoryStorage](p.ctx); history != nil {
	} else if clashServer := p.router.ClashServer(); clashServer != nil {
		history = clashServer.HistoryStorage()
	} else {
		history = urltest.NewHistoryStorage()
	}
	p.healchcheckHistory = history
	if p.ticker != nil {
		return
	}
	p.access.Lock()
	defer p.access.Unlock()
	if p.ticker != nil {
		return
	}
	interval, _ := time.ParseDuration("1m")
	p.ticker = time.NewTicker(interval)
	go p.loopCheck()
}

func (p *HTTPProvider) Close() error {
	if p.ticker == nil {
		return nil
	}
	p.ticker.Stop()
	close(p.close)
	return nil
}

func (p *HTTPProvider) loopCheck() {
	go p.CheckOutbounds(true)
	go p.UpdateProvider(p.ctx, p.router, false)
	for {
		p.pauseManager.WaitActive()
		select {
		case <-p.close:
			return
		case <-p.ticker.C:
			p.CheckOutbounds(false)
			p.UpdateProvider(p.ctx, p.router, false)
		}
	}
}

func (p *HTTPProvider) FetchHTTP(httpClient *http.Client, parsedURL *url.URL) (string, string, error) {
	request, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		return "", "", err
	}
	request.Header.Add("User-Agent", p.ua)
	response, err := httpClient.Do(request)
	if err != nil {
		return "", "", err
	}
	defer response.Body.Close()
	contentRaw, err := io.ReadAll(response.Body)
	if err != nil {
		return "", "", err
	}
	if len(contentRaw) == 0 {
		return "", "", E.New("empty response")
	}
	content := string(contentRaw)
	subInfo := response.Header.Get("subscription-userinfo")
	if subInfo != "" {
		subInfo = "# " + subInfo + ";"
	}
	return content, subInfo, nil
}

func (p *HTTPProvider) FetchContent(router adapter.Router) (string, string, error) {
	detour := router.DefaultOutboundForConnection()
	if p.detour != "" {
		if outbound, ok := router.Outbound(p.detour); ok {
			detour = outbound
		}
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return detour.DialContext(ctx, network, M.ParseSocksaddr(addr))
			},
			ForceAttemptHTTP2: true,
		},
	}
	defer httpClient.CloseIdleConnections()
	parsedURL, err := url.Parse(p.url)
	if err != nil {
		return "", "", err
	}
	switch parsedURL.Scheme {
	case "":
		parsedURL.Scheme = "http"
		fallthrough
	case "http", "https":
		content, subInfo, err := p.FetchHTTP(httpClient, parsedURL)
		if err != nil {
			return "", "", err
		}
		return content, subInfo, nil
	default:
		return "", "", E.New("invalid url scheme")
	}
}

func (p *HTTPProvider) ContentFromHTTP(router adapter.Router) (string, error) {
	content, subInfo, err := p.FetchContent(router)
	if err != nil {
		return "", E.Cause(err, "fetch provider ", p.tag, " failed")
	}
	if content == "" {
		return "", E.New("fetch provider ", p.tag, " failed: empty content")
	}
	path := p.path
	if !p.ParseSubInfo(subInfo) {
		firstLine, others := GetFirstLine(content)
		if p.ParseSubInfo(firstLine) {
			content = others
			upload := p.subscriptionInfo.upload
			download := p.subscriptionInfo.download
			total := p.subscriptionInfo.total
			expire := p.subscriptionInfo.expire
			subInfo = fmt.Sprint("# upload=", upload, "; download=", download, "; total=", total, "; expire=", expire, ";")
		}
	}
	p.updateTime = time.Now()
	content = ParseContent(content)
	fileContent := content
	if subInfo != "" {
		fileContent = subInfo + "\n" + fileContent
	}
	os.WriteFile(path, []byte(fileContent), 0o666)
	return content, nil
}

func (p *HTTPProvider) GetContent(router adapter.Router) (string, error) {
	if !p.start {
		p.start = true
		return p.GetContentFromFile(router), nil
	}
	return p.ContentFromHTTP(router)
}

func (p *HTTPProvider) ParseProvider(ctx context.Context, router adapter.Router) error {
	content, err := p.GetContent(router)
	if err != nil {
		return err
	}
	return p.ParseOutbounds(ctx, router, content)
}

func (p *HTTPProvider) UpdateProvider(ctx context.Context, router adapter.Router, force bool) error {
	p.access.Lock()
	defer p.access.Unlock()
	if p.updating.Swap(true) {
		return E.New("provider is updating")
	}
	defer p.updating.Store(false)
	if !force && time.Since(p.lastDownload) < p.interval {
		return nil
	}
	p.lastDownload = time.Now()
	p.RLock()
	defer p.RUnlock()
	outboundsBackup, outboundByTagBackup, subscriptionInfoBackup := p.BackupProvider()
	err := p.RunFuncsWithRevert(
		func() error { return p.ParseProvider(ctx, router) },
		func() error { return p.StartOutbounds(router) },
		func() error { return p.UpdateGroups(router) },
	)
	if err != nil {
		p.RevertProvider(outboundsBackup, outboundByTagBackup, subscriptionInfoBackup)
	}
	p.CheckOutbounds(true)
	return nil
}

func NewHTTPProvider(ctx context.Context, router adapter.Router, logger log.ContextLogger, options option.OutboundProvider, path string) (*HTTPProvider, error) {
	httpOptions := options.HTTPOptions
	healthcheckInterval := time.Duration(options.HealthcheckInterval)
	url := httpOptions.Url
	ua := httpOptions.UserAgent
	downloadInterval := time.Duration(options.HTTPOptions.Interval)
	defaultDownloadInterval, _ := time.ParseDuration("1h")
	if url == "" {
		return nil, E.New("provider download url missing")
	}
	if ua == "" {
		ua = "sing-box"
	}
	if healthcheckInterval == 0 {
		healthcheckInterval = C.DefaultURLTestInterval
	}
	if downloadInterval < defaultDownloadInterval {
		downloadInterval = defaultDownloadInterval
	}
	provider := &HTTPProvider{
		myProviderAdapter: myProviderAdapter{
			ctx:                 ctx,
			router:              router,
			logger:              logger,
			tag:                 options.Tag,
			path:                path,
			healthcheckUrl:      options.HealthcheckUrl,
			healthcheckInterval: healthcheckInterval,
			lastHealthcheck:     time.Unix(int64(0), int64(0)),
			providerType:        C.TypeHTTPProvider,
			overrideDialer:      options.OverrideDialer,
			updateTime:          time.Unix(int64(0), int64(0)),
			close:               make(chan struct{}),
			pauseManager:        pause.ManagerFromContext(ctx),
			subscriptionInfo: SubscriptionInfo{
				upload:   0,
				download: 0,
				total:    0,
				expire:   0,
			},
			outbounds:     []adapter.Outbound{},
			outboundByTag: make(map[string]adapter.Outbound),
		},
		url:          httpOptions.Url,
		ua:           ua,
		interval:     downloadInterval,
		lastDownload: time.Unix(int64(0), int64(0)),
		detour:       httpOptions.Detour,
		start:        false,
	}
	err := provider.ParseProvider(ctx, router)
	if err != nil {
		return nil, err
	}
	provider.lastDownload = provider.updateTime
	return provider, nil
}
