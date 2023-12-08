package provider

import (
	"context"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/urltest"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/service"
	"github.com/sagernet/sing/service/pause"
)

var _ adapter.OutboundProvider = (*FileProvider)(nil)

type FileProvider struct {
	myProviderAdapter
}

func (p *FileProvider) Start() {
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

func (p *FileProvider) Close() error {
	if p.ticker == nil {
		return nil
	}
	p.ticker.Stop()
	close(p.close)
	return nil
}

func (p *FileProvider) loopCheck() {
	go p.CheckOutbounds(true)
	for {
		p.pauseManager.WaitActive()
		select {
		case <-p.close:
			return
		case <-p.ticker.C:
			p.CheckOutbounds(false)
		}
	}
}

func (p *FileProvider) ParseProvider(ctx context.Context, router adapter.Router) error {
	content := ParseContent(p.GetContentFromFile(router))
	return p.ParseOutbounds(ctx, router, content)
}

func (p *FileProvider) UpdateProvider(ctx context.Context, router adapter.Router, force bool) error {
	p.access.Lock()
	defer p.access.Unlock()
	if p.updating.Swap(true) {
		return E.New("provider is updating")
	}
	defer p.updating.Store(false)
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

func NewFileProvider(ctx context.Context, router adapter.Router, logger log.ContextLogger, options option.OutboundProvider, path string) (*FileProvider, error) {
	interval := time.Duration(options.HealthcheckInterval)
	if interval == 0 {
		interval = C.DefaultURLTestInterval
	}
	provider := &FileProvider{
		myProviderAdapter: myProviderAdapter{
			ctx:                 ctx,
			router:              router,
			logger:              logger,
			tag:                 options.Tag,
			path:                path,
			healthcheckUrl:      options.HealthcheckUrl,
			healthcheckInterval: interval,
			overrideDialer:      options.OverrideDialer,
			lastHealthcheck:     time.Unix(int64(0), int64(0)),
			providerType:        C.TypeFileProvider,
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
	}
	err := provider.ParseProvider(ctx, router)
	if err != nil {
		return nil, err
	}
	return provider, nil
}
