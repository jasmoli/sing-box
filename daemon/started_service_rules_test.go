package daemon

import (
	"testing"

	"github.com/sagernet/sing-box/adapter"
)

type testRule struct {
	adapter.Rule
	ruleType string
	payload  string
	proxy    string
	disabled bool
	uuid     string
}

func (r *testRule) Type() string {
	return r.ruleType
}

func (r *testRule) String() string {
	return r.payload
}

func (r *testRule) Action() adapter.RuleAction {
	return testRuleAction{actionType: r.proxy}
}

func (r *testRule) Disabled() bool {
	return r.disabled
}

func (r *testRule) UUID() string {
	return r.uuid
}

type testRuleAction struct {
	actionType string
}

func (a testRuleAction) Type() string {
	return a.actionType
}

func (a testRuleAction) String() string {
	return a.actionType
}

func TestRuleInfo(t *testing.T) {
	rule := &testRule{
		ruleType: "default",
		payload:  "domain_suffix=example.com",
		proxy:    "proxy",
		disabled: true,
		uuid:     "rule-uuid",
	}
	info := ruleInfo(rule)
	if info.Type != "default" || info.Payload != "domain_suffix=example.com" {
		t.Fatalf("unexpected rule info: %+v", info)
	}
	if info.Proxy != "proxy" || !info.Disabled || info.Uuid != "rule-uuid" {
		t.Fatalf("unexpected rule info: %+v", info)
	}
}
