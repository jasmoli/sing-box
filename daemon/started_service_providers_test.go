package daemon

import (
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/urltest"
)

type testProvider struct {
	adapter.Provider
	tag          string
	providerType string
	outbounds    []adapter.Outbound
	updatedAt    time.Time
	subscription adapter.SubscriptionInfo
}

func (p *testProvider) Tag() string {
	return p.tag
}

func (p *testProvider) Type() string {
	return p.providerType
}

func (p *testProvider) Outbounds() []adapter.Outbound {
	return p.outbounds
}

func (p *testProvider) UpdatedAt() time.Time {
	return p.updatedAt
}

func (p *testProvider) SubscriptionInfo() adapter.SubscriptionInfo {
	return p.subscription
}

type testOutbound struct {
	adapter.Outbound
	tag          string
	outboundType string
}

func (o *testOutbound) Tag() string {
	return o.tag
}

func (o *testOutbound) Type() string {
	return o.outboundType
}

type testRuleSet struct {
	adapter.RuleSet
	name      string
	ruleType  string
	format    string
	ruleCount uint64
	updatedAt time.Time
}

func (r *testRuleSet) Name() string {
	return r.name
}

func (r *testRuleSet) Type() string {
	return r.ruleType
}

func (r *testRuleSet) Format() string {
	return r.format
}

func (r *testRuleSet) RuleCount() uint64 {
	return r.ruleCount
}

func (r *testRuleSet) UpdatedAt() time.Time {
	return r.updatedAt
}

func TestProviderInfo(t *testing.T) {
	historyStorage := urltest.NewHistoryStorage()
	t.Cleanup(func() {
		_ = historyStorage.Close()
	})
	historyStorage.StoreURLTestHistory("node", &adapter.URLTestHistory{
		Time:  time.Unix(1700000000, 0),
		Delay: 123,
	})
	instance := &Instance{urlTestHistoryStorage: historyStorage}
	provider := &testProvider{
		tag:          "subscription",
		providerType: "remote",
		outbounds: []adapter.Outbound{
			&testOutbound{tag: "node", outboundType: "shadowsocks"},
		},
		updatedAt: time.Unix(1700000001, 0),
		subscription: adapter.SubscriptionInfo{
			Upload:   1,
			Download: 2,
			Total:    3,
			Expire:   4,
		},
	}
	info := providerInfo(instance, provider)
	if info.Tag != "subscription" {
		t.Fatalf("tag = %q", info.Tag)
	}
	if info.VehicleType != "HTTP" {
		t.Fatalf("vehicle type = %q", info.VehicleType)
	}
	if info.UpdatedAt != 1700000001 {
		t.Fatalf("updated at = %d", info.UpdatedAt)
	}
	if len(info.Outbounds) != 1 {
		t.Fatalf("outbounds = %d", len(info.Outbounds))
	}
	if info.Outbounds[0].Tag != "node" || info.Outbounds[0].Type != "shadowsocks" {
		t.Fatalf("unexpected outbound: %+v", info.Outbounds[0])
	}
	if info.Outbounds[0].UrlTestTime != 1700000000 || info.Outbounds[0].UrlTestDelay != 123 {
		t.Fatalf("unexpected URL test result: %+v", info.Outbounds[0])
	}
	if info.SubscriptionInfo == nil || info.SubscriptionInfo.Total != 3 {
		t.Fatalf("unexpected subscription info: %+v", info.SubscriptionInfo)
	}
}

func TestRuleProviderInfo(t *testing.T) {
	ruleSet := &testRuleSet{
		name:      "rules",
		ruleType:  "remote",
		format:    "binary",
		ruleCount: 42,
		updatedAt: time.Unix(1700000002, 0),
	}
	info := ruleProviderInfo(ruleSet)
	if info.Tag != "rules" || info.VehicleType != "HTTP" || info.Behavior != "Binary" {
		t.Fatalf("unexpected rule provider info: %+v", info)
	}
	if info.RuleCount != 42 || info.UpdatedAt != 1700000002 {
		t.Fatalf("unexpected rule provider info: %+v", info)
	}
}
