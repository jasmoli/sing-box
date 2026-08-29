package remote

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/provider"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/provider/parser"
	"github.com/sagernet/sing/common"
	E "github.com/sagernet/sing/common/exceptions"
	F "github.com/sagernet/sing/common/format"
	"github.com/sagernet/sing/common/json"
	"github.com/sagernet/sing/service"
	"github.com/sagernet/sing/service/filemanager"
)

func RegisterProviderRemote(registry *provider.Registry) {
	provider.Register[option.ProviderRemoteOptions](registry, C.ProviderTypeRemote, NewProviderRemote)
}

var _ adapter.Provider = (*ProviderRemote)(nil)

type ProviderRemote struct {
	provider.Adapter
	ctx              context.Context
	cancel           context.CancelFunc
	logger           log.ContextLogger
	outbound         adapter.OutboundManager
	cacheFile        adapter.CacheFile
	httpClient       adapter.HTTPTransport
	httpClientMgr    adapter.HTTPClientManager
	httpClientOpts   *option.HTTPClientOptions
	downloadDetour   string
	override         *option.ProviderOverrideOptions
	path             string
	url              string
	userAgent        string
	updateInterval   time.Duration
	exclude          *regexp.Regexp
	include          *regexp.Regexp
	lastEtag         string
	lastOutOpts      []option.Outbound
	lastUpdated      time.Time
	subscriptionInfo adapter.SubscriptionInfo
	ticker           *time.Ticker
	updating         atomic.Bool
}

func NewProviderRemote(ctx context.Context, router adapter.Router, logFactory log.Factory, tag string, options option.ProviderRemoteOptions) (adapter.Provider, error) {
	if options.URL == "" {
		return nil, E.New("provider URL is required")
	}
	updateInterval := time.Duration(options.UpdateInterval)
	if updateInterval <= 0 {
		updateInterval = 24 * time.Hour
	}
	if updateInterval < time.Minute {
		updateInterval = time.Minute
	}
	var userAgent string
	if options.UserAgent == "" {
		userAgent = "sing-box " + C.Version
	} else {
		userAgent = options.UserAgent
	}
	outbound := service.FromContext[adapter.OutboundManager](ctx)
	logger := logFactory.NewLogger(F.ToString("provider/remote", "[", tag, "]"))
	ctx, cancel := context.WithCancel(ctx)
	httpClientManager := service.FromContext[adapter.HTTPClientManager](ctx)
	var filePath string
	if options.Path != "" {
		filePath = filemanager.BasePath(ctx, options.Path)
		filePath, _ = filepath.Abs(filePath)
	}
	return &ProviderRemote{
		Adapter:        provider.NewAdapter(ctx, router, outbound, logFactory, logger, tag, C.ProviderTypeRemote, options.HealthCheck),
		ctx:            ctx,
		cancel:         cancel,
		logger:         logger,
		outbound:       outbound,
		httpClientMgr:  httpClientManager,
		httpClientOpts: options.HTTPClient,
		override:       options.Override,
		downloadDetour: options.DownloadDetour,
		path:           filePath,
		url:            options.URL,
		userAgent:      userAgent,
		updateInterval: updateInterval,
		exclude:        (*regexp.Regexp)(options.Exclude),
		include:        (*regexp.Regexp)(options.Include),
	}, nil
}

func (s *ProviderRemote) Start() error {
	if s.httpClientMgr != nil {
		var err error
		s.httpClient, err = s.resolveTransport()
		if err != nil {
			return E.Cause(err, "create provider http client")
		}
	}
	if s.httpClient == nil {
		return E.New("http client transport is not initialized")
	}
	var loaded bool
	if s.path != "" {
		loaded = s.loadFromFile()
	} else {
		s.cacheFile = service.FromContext[adapter.CacheFile](s.ctx)
		loaded = s.loadFromCache()
	}
	if loaded {
		s.UpdateGroups()
	}
	go s.loopUpdate()
	return s.Adapter.Start()
}

func (s *ProviderRemote) resolveTransport() (adapter.HTTPTransport, error) {
	if s.httpClientOpts != nil && !s.httpClientOpts.IsEmpty() {
		if s.downloadDetour != "" {
			return nil, E.New("http_client is conflict with download_detour field")
		}
		return s.httpClientMgr.ResolveTransport(s.ctx, s.logger, *s.httpClientOpts)
	}
	if s.downloadDetour != "" {
		return s.httpClientMgr.ResolveTransport(s.ctx, s.logger, option.HTTPClientOptions{
			DialerOptions: option.DialerOptions{
				Detour: s.downloadDetour,
			},
			DisableEmptyDirectCheck: true,
		})
	}
	defaultTransport := s.httpClientMgr.DefaultTransport()
	if defaultTransport == nil {
		return nil, E.New("default http client transport is not initialized")
	}
	return defaultTransport, nil
}

func (s *ProviderRemote) loadFromFile() bool {
	file, err := filemanager.OpenFile(s.ctx, s.path, os.O_RDONLY, 0)
	if err != nil {
		return false
	}
	content, err := io.ReadAll(file)
	fileInfo, statErr := file.Stat()
	closeErr := file.Close()
	if err != nil || statErr != nil || closeErr != nil {
		return false
	}
	s.lastUpdated = fileInfo.ModTime()
	contentStr := parser.DecodeBase64Safe(string(content))
	firstLine, others := parser.GetFirstLine(contentStr)
	if info, ok := provider.ParseInfo(firstLine); ok {
		s.subscriptionInfo = info
		contentStr = parser.DecodeBase64Safe(others)
	}
	if err := s.updateProviderFromContent(contentStr); err != nil {
		s.logger.Error(E.Cause(err, "restore outbound provider from file"))
		return false
	}
	return true
}

func (s *ProviderRemote) loadFromCache() bool {
	if s.cacheFile == nil {
		return false
	}
	saveSub := s.cacheFile.LoadSubscription(s.Tag())
	if saveSub == nil {
		return false
	}
	content, _ := parser.DecodeBase64URLSafe(string(saveSub.Content))
	firstLine, others := parser.GetFirstLine(content)
	if info, ok := provider.ParseInfo(firstLine); ok {
		s.subscriptionInfo = info
		content, _ = parser.DecodeBase64URLSafe(others)
	}
	if err := s.updateProviderFromContent(content); err != nil {
		s.logger.Error(E.Cause(err, "restore cached outbound provider"))
		return false
	}
	s.lastUpdated, s.lastEtag = saveSub.LastUpdated, saveSub.LastEtag
	return true
}

func (s *ProviderRemote) Update() error {
	if s.ticker != nil {
		s.ticker.Reset(s.updateInterval)
	}
	return s.fetch(s.ctx)
}

func (s *ProviderRemote) UpdatedAt() time.Time {
	return s.lastUpdated
}

func (s *ProviderRemote) SubscriptionInfo() adapter.SubscriptionInfo {
	return s.subscriptionInfo
}

func (s *ProviderRemote) Close() error {
	s.cancel()
	if s.ticker != nil {
		s.ticker.Stop()
	}
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return common.Close(&s.Adapter)
}

func (s *ProviderRemote) updateOnce() {
	if err := s.fetch(s.ctx); err != nil {
		s.logger.Error("update outbound provider: ", err)
	}
}

func (s *ProviderRemote) fetch(ctx context.Context) error {
	if s.updating.Swap(true) {
		return E.New("provider is updating")
	}
	defer s.updating.Store(false)
	s.logger.Debug("updating outbound provider ", s.Tag(), " from URL: ", s.url)
	request, err := http.NewRequest(http.MethodGet, s.url, nil)
	if err != nil {
		return err
	}
	if s.lastEtag != "" {
		request.Header.Set("If-None-Match", s.lastEtag)
	}
	request.Header.Set("User-Agent", s.userAgent)
	response, err := s.httpClient.RoundTrip(request.WithContext(ctx))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	infoStr := response.Header.Get("subscription-userinfo")
	info, hasInfo := provider.ParseInfo(infoStr)
	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusNotModified:
		s.subscriptionInfo = info
		s.lastUpdated = time.Now()
		s.persistNotModified(infoStr, hasInfo)
		s.logger.Info("update outbound provider ", s.Tag(), ": not modified")
		return nil
	default:
		return E.New("unexpected status: ", response.Status)
	}
	contentRaw, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	eTagHeader := response.Header.Get("Etag")
	if eTagHeader != "" {
		s.lastEtag = eTagHeader
	}
	content, _ := parser.DecodeBase64URLSafe(string(contentRaw))
	if !hasInfo {
		firstLine, others := parser.GetFirstLine(content)
		if info, hasInfo = provider.ParseInfo(firstLine); hasInfo {
			infoStr = firstLine
			content, _ = parser.DecodeBase64URLSafe(others)
		}
	}
	if err := s.updateProviderFromContent(content); err != nil {
		return err
	}
	s.UpdateGroups()
	s.subscriptionInfo = info
	s.lastUpdated = time.Now()
	s.persistContent(infoStr, hasInfo)
	s.logger.Info("updated outbound provider ", s.Tag())
	return nil
}

func formatInfoLine(info adapter.SubscriptionInfo) string {
	return fmt.Sprintf("# upload=%d; download=%d; total=%d; expire=%d;",
		info.Upload, info.Download, info.Total, info.Expire)
}

// persistContent saves the parsed outbounds to the path file (when set) or the
// cache file.
func (s *ProviderRemote) persistContent(infoStr string, hasInfo bool) {
	content, _ := json.Marshal(option.Options{
		Outbounds: s.lastOutOpts,
	})
	if hasInfo {
		line := infoStr
		if s.path != "" && !strings.HasPrefix(line, "#") {
			line = formatInfoLine(s.subscriptionInfo)
		}
		content = append([]byte(line+"\n"), content...)
	}
	if s.path != "" {
		if err := filemanager.WriteFile(s.ctx, s.path, content, 0o666); err != nil {
			s.logger.Error("save outbound provider file: ", err)
		}
		return
	}
	if s.cacheFile != nil {
		err := s.cacheFile.SaveSubscription(s.Tag(), &adapter.SavedBinary{
			LastUpdated: s.lastUpdated,
			Content:     content,
			LastEtag:    s.lastEtag,
		})
		if err != nil {
			s.logger.Error("save outbound provider cache file: ", err)
		}
	}
}

// persistNotModified refreshes the persisted subscription info and update time
// on a 304 response.
func (s *ProviderRemote) persistNotModified(infoStr string, hasInfo bool) {
	var saved *adapter.SavedBinary
	if s.path != "" {
		content, err := filemanager.ReadFile(s.ctx, s.path)
		if err != nil {
			return
		}
		saved = &adapter.SavedBinary{Content: content}
	} else if s.cacheFile != nil {
		saved = s.cacheFile.LoadSubscription(s.Tag())
		if saved == nil {
			return
		}
	} else {
		return
	}
	if !hasInfo {
		saved.LastUpdated = s.lastUpdated
		s.persistSavedBinary(saved)
		return
	}
	line := infoStr
	if s.path != "" && !strings.HasPrefix(line, "#") {
		line = formatInfoLine(s.subscriptionInfo)
	}
	contentStr := parser.DecodeBase64Safe(string(saved.Content))
	firstLine, others := parser.GetFirstLine(contentStr)
	if _, ok := provider.ParseInfo(firstLine); ok {
		contentStr = others
	}
	saved.Content = append([]byte(line+"\n"), []byte(contentStr)...)
	saved.LastUpdated = s.lastUpdated
	s.persistSavedBinary(saved)
}

func (s *ProviderRemote) persistSavedBinary(saved *adapter.SavedBinary) {
	if s.path != "" {
		if err := filemanager.WriteFile(s.ctx, s.path, saved.Content, 0o666); err != nil {
			s.logger.Error("save outbound provider file: ", err)
		}
		return
	}
	if s.cacheFile != nil {
		if err := s.cacheFile.SaveSubscription(s.Tag(), saved); err != nil {
			s.logger.Error("save outbound provider cache file: ", err)
		}
	}
}

func (s *ProviderRemote) loopUpdate() {
	if time.Since(s.lastUpdated) < s.updateInterval {
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(time.Until(s.lastUpdated.Add(s.updateInterval))):
			s.updateOnce()
		}
	} else {
		s.updateOnce()
	}
	s.ticker = time.NewTicker(s.updateInterval)
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.ticker.C:
			s.updateOnce()
		}
	}
}

func (s *ProviderRemote) updateProviderFromContent(content string) error {
	outboundOpts, err := parser.ParseSubscription(s.ctx, content, s.override)
	if err != nil {
		return err
	}
	outboundOpts = common.Filter(outboundOpts, func(it option.Outbound) bool {
		return (s.exclude == nil || !s.exclude.MatchString(it.Tag)) && (s.include == nil || s.include.MatchString(it.Tag))
	})
	s.UpdateOutbounds(s.lastOutOpts, outboundOpts)
	s.lastOutOpts = outboundOpts
	return nil
}
