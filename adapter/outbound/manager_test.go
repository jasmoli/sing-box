package outbound

import (
	"context"
	"net"
	"os"
	"testing"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/endpoint"
	"github.com/sagernet/sing-box/log"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/stretchr/testify/require"
)

type testLifecycleOutbound struct {
	tag    string
	stages []adapter.StartStage
}

func (o *testLifecycleOutbound) Type() string           { return "test" }
func (o *testLifecycleOutbound) Tag() string            { return o.tag }
func (o *testLifecycleOutbound) Network() []string      { return []string{N.NetworkTCP} }
func (o *testLifecycleOutbound) Dependencies() []string { return nil }

func (o *testLifecycleOutbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	return nil, os.ErrInvalid
}

func (o *testLifecycleOutbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	return nil, os.ErrInvalid
}

func (o *testLifecycleOutbound) Start(stage adapter.StartStage, scope *adapter.Scope) error {
	o.stages = append(o.stages, stage)
	return nil
}

func TestCreateStartsLifecycle(t *testing.T) {
	registry := NewRegistry()
	created := &testLifecycleOutbound{tag: "test"}
	Register[struct{}](registry, "test", func(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options struct{}) (adapter.Outbound, error) {
		return created, nil
	})
	manager := NewManager(registry, endpoint.NewManager(nil), "")
	scope := adapter.NewScope(context.Background(), log.NewNOPFactory().Logger())
	require.NoError(t, manager.Start(adapter.StartStateStart, scope))
	require.NoError(t, manager.Create(context.Background(), nil, log.NewNOPFactory().Logger(), "test", "test", nil))
	require.Equal(t, []adapter.StartStage{
		adapter.StartStateInitialize,
		adapter.StartStateStart,
	}, created.stages)
}
