package daemon

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/urltest"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/protocol/group"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const providerHealthCheckDefaultTimeout = 5 * time.Second

func (s *StartedService) startedInstance() (*Instance, error) {
	s.serviceAccess.RLock()
	defer s.serviceAccess.RUnlock()
	if s.serviceStatus.Status != ServiceStatus_STARTED {
		return nil, os.ErrInvalid
	}
	return s.instance, nil
}

func (s *StartedService) GetProviders(ctx context.Context, empty *emptypb.Empty) (*Providers, error) {
	instance, err := s.startedInstance()
	if err != nil {
		return nil, err
	}
	if instance.providerManager == nil {
		return &Providers{}, nil
	}
	providers := instance.providerManager.Providers()
	response := &Providers{
		Providers: make([]*Provider, 0, len(providers)),
	}
	for _, provider := range providers {
		response.Providers = append(response.Providers, providerInfo(instance, provider))
	}
	return response, nil
}

func (s *StartedService) GetProvider(ctx context.Context, request *ProviderRequest) (*Provider, error) {
	instance, err := s.startedInstance()
	if err != nil {
		return nil, err
	}
	provider, loaded := findProvider(instance, request.Tag)
	if !loaded {
		return nil, status.Error(codes.NotFound, "provider not found: "+request.Tag)
	}
	return providerInfo(instance, provider), nil
}

func (s *StartedService) UpdateProvider(ctx context.Context, request *ProviderRequest) (*emptypb.Empty, error) {
	instance, err := s.startedInstance()
	if err != nil {
		return nil, err
	}
	provider, loaded := findProvider(instance, request.Tag)
	if !loaded {
		return nil, status.Error(codes.NotFound, "provider not found: "+request.Tag)
	}
	if updater, isUpdater := provider.(adapter.ProviderUpdater); isUpdater {
		if updateErr := updater.Update(); updateErr != nil {
			return nil, status.Error(codes.Unavailable, updateErr.Error())
		}
	}
	return &emptypb.Empty{}, nil
}

func (s *StartedService) HealthCheckProvider(ctx context.Context, request *ProviderRequest) (*ProviderHealthCheckResult, error) {
	instance, err := s.startedInstance()
	if err != nil {
		return nil, err
	}
	provider, loaded := findProvider(instance, request.Tag)
	if !loaded {
		return nil, status.Error(codes.NotFound, "provider not found: "+request.Tag)
	}
	delays, checkErr := provider.HealthCheck(ctx)
	if checkErr != nil {
		return nil, status.Error(codes.Unavailable, checkErr.Error())
	}
	result := &ProviderHealthCheckResult{
		Delays: make(map[string]int32, len(delays)),
	}
	for tag, delay := range delays {
		result.Delays[tag] = int32(delay)
	}
	return result, nil
}

func (s *StartedService) HealthCheckProviderOutbound(ctx context.Context, request *ProviderOutboundRequest) (*ProviderHealthCheckResult, error) {
	instance, err := s.startedInstance()
	if err != nil {
		return nil, err
	}
	provider, loaded := findProvider(instance, request.Tag)
	if !loaded {
		return nil, status.Error(codes.NotFound, "provider not found: "+request.Tag)
	}
	outbound, loaded := provider.Outbound(request.OutboundTag)
	if !loaded {
		return nil, status.Error(codes.NotFound, "outbound not found in provider: "+request.OutboundTag)
	}
	link := request.Url
	if strings.HasPrefix(link, "http://") {
		link = ""
	}
	timeout := time.Duration(request.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = providerHealthCheckDefaultTimeout
	}
	testCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	delay, testErr := urltest.URLTest(testCtx, link, outbound)
	realTag := group.RealTag(instance.outboundManager, outbound)
	if testErr != nil {
		instance.urlTestHistoryStorage.DeleteURLTestHistory(realTag)
	} else {
		instance.urlTestHistoryStorage.StoreURLTestHistory(realTag, &adapter.URLTestHistory{
			Time:  time.Now(),
			Delay: delay,
		})
	}
	if testCtx.Err() != nil {
		return nil, status.Error(codes.DeadlineExceeded, testCtx.Err().Error())
	}
	if testErr != nil || delay == 0 {
		return nil, status.Error(codes.Unavailable, "delay test failed")
	}
	return &ProviderHealthCheckResult{
		Delays: map[string]int32{
			request.OutboundTag: int32(delay),
		},
	}, nil
}

func (s *StartedService) GetRuleProviders(ctx context.Context, empty *emptypb.Empty) (*RuleProviders, error) {
	instance, err := s.startedInstance()
	if err != nil {
		return nil, err
	}
	if instance.router == nil {
		return &RuleProviders{}, nil
	}
	ruleSets := instance.router.RuleSets()
	response := &RuleProviders{
		Providers: make([]*RuleProvider, 0, len(ruleSets)),
	}
	for _, ruleSet := range ruleSets {
		response.Providers = append(response.Providers, ruleProviderInfo(ruleSet))
	}
	return response, nil
}

func (s *StartedService) GetRuleProvider(ctx context.Context, request *ProviderRequest) (*RuleProvider, error) {
	instance, err := s.startedInstance()
	if err != nil {
		return nil, err
	}
	if instance.router == nil {
		return nil, status.Error(codes.NotFound, "rule provider not found: "+request.Tag)
	}
	ruleSet, loaded := instance.router.RuleSet(request.Tag)
	if !loaded {
		return nil, status.Error(codes.NotFound, "rule provider not found: "+request.Tag)
	}
	return ruleProviderInfo(ruleSet), nil
}

func (s *StartedService) UpdateRuleProvider(ctx context.Context, request *ProviderRequest) (*emptypb.Empty, error) {
	instance, err := s.startedInstance()
	if err != nil {
		return nil, err
	}
	if instance.router == nil {
		return nil, status.Error(codes.NotFound, "rule provider not found: "+request.Tag)
	}
	ruleSet, loaded := instance.router.RuleSet(request.Tag)
	if !loaded {
		return nil, status.Error(codes.NotFound, "rule provider not found: "+request.Tag)
	}
	if updater, isUpdater := ruleSet.(adapter.ProviderUpdater); isUpdater {
		if updateErr := updater.Update(); updateErr != nil {
			return nil, status.Error(codes.Unavailable, updateErr.Error())
		}
	}
	return &emptypb.Empty{}, nil
}

func findProvider(instance *Instance, tag string) (adapter.Provider, bool) {
	if instance.providerManager == nil {
		return nil, false
	}
	return instance.providerManager.Get(tag)
}

func providerInfo(instance *Instance, provider adapter.Provider) *Provider {
	info := &Provider{
		Tag:         provider.Tag(),
		VehicleType: C.ProviderDisplayName(provider.Type()),
	}
	updatedAt := provider.UpdatedAt()
	if !updatedAt.IsZero() {
		info.UpdatedAt = updatedAt.Unix()
	}
	for _, outbound := range provider.Outbounds() {
		item := &GroupItem{
			Tag:  outbound.Tag(),
			Type: outbound.Type(),
		}
		history := instance.urlTestHistoryStorage.LoadURLTestHistory(group.RealTag(instance.outboundManager, outbound))
		if history != nil {
			item.UrlTestTime = history.Time.Unix()
			item.UrlTestDelay = int32(history.Delay)
		}
		info.Outbounds = append(info.Outbounds, item)
	}
	if providerWithInfo, isProviderWithInfo := provider.(adapter.ProviderSubscriptionInfo); isProviderWithInfo {
		subscription := providerWithInfo.SubscriptionInfo()
		info.SubscriptionInfo = &SubscriptionInfo{
			Upload:   subscription.Upload,
			Download: subscription.Download,
			Total:    subscription.Total,
			Expire:   subscription.Expire,
		}
	}
	return info
}

func ruleProviderInfo(ruleSet adapter.RuleSet) *RuleProvider {
	info := &RuleProvider{
		Tag:         ruleSet.Name(),
		VehicleType: C.ProviderDisplayName(ruleSet.Type()),
		Behavior:    C.RuleSetFormatDisplayName(ruleSet.Format()),
		RuleCount:   ruleSet.RuleCount(),
	}
	updatedAt := ruleSet.UpdatedAt()
	if !updatedAt.IsZero() {
		info.UpdatedAt = updatedAt.Unix()
	}
	return info
}
