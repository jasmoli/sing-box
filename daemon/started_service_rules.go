package daemon

import (
	"context"

	"github.com/sagernet/sing-box/adapter"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *StartedService) GetRules(ctx context.Context, empty *emptypb.Empty) (*Rules, error) {
	instance, err := s.startedInstance()
	if err != nil {
		return nil, err
	}
	if instance.router == nil {
		return &Rules{}, nil
	}
	rawRules := instance.router.Rules()
	response := &Rules{
		Rules: make([]*Rule, 0, len(rawRules)),
	}
	for _, rule := range rawRules {
		response.Rules = append(response.Rules, ruleInfo(rule))
	}
	return response, nil
}

func (s *StartedService) ChangeRuleStatus(ctx context.Context, request *ChangeRuleStatusRequest) (*emptypb.Empty, error) {
	instance, err := s.startedInstance()
	if err != nil {
		return nil, err
	}
	if instance.router == nil {
		return nil, status.Error(codes.NotFound, "rule not found: "+request.Uuid)
	}
	rule, loaded := instance.router.Rule(request.Uuid)
	if !loaded {
		return nil, status.Error(codes.NotFound, "rule not found: "+request.Uuid)
	}
	rule.ChangeStatus()
	return &emptypb.Empty{}, nil
}

func ruleInfo(rule adapter.Rule) *Rule {
	return &Rule{
		Type:     rule.Type(),
		Payload:  rule.String(),
		Proxy:    rule.Action().String(),
		Disabled: rule.Disabled(),
		Uuid:     rule.UUID(),
	}
}
