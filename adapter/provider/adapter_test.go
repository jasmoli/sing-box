package provider

import (
	"context"
	"net"
	"os"
	"testing"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/endpoint"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/stretchr/testify/require"
)

type testOutboundOptions struct {
	Marker string `json:"marker"`
}

type testOutbound struct {
	tag    string
	marker string
}

func (o *testOutbound) Type() string           { return "test" }
func (o *testOutbound) Tag() string            { return o.tag }
func (o *testOutbound) Network() []string      { return []string{N.NetworkTCP} }
func (o *testOutbound) Dependencies() []string { return nil }

func (o *testOutbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	return nil, os.ErrInvalid
}

func (o *testOutbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	return nil, os.ErrInvalid
}

func TestUpdateOutboundsReplacesChangedOutbound(t *testing.T) {
	ctx := context.Background()
	logFactory := log.NewNOPFactory()
	outboundRegistry := outbound.NewRegistry()
	outbound.Register[testOutboundOptions](outboundRegistry, "test", func(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options testOutboundOptions) (adapter.Outbound, error) {
		return &testOutbound{tag: tag, marker: options.Marker}, nil
	})
	manager := outbound.NewManager(outboundRegistry, endpoint.NewManager(nil), "")
	adapterProvider := NewAdapter(ctx, nil, manager, logFactory, logFactory.Logger(), "test", "test", option.ProviderHealthCheckOptions{})

	oldOpts := []option.Outbound{{Tag: "node", Type: "test", Options: &testOutboundOptions{Marker: "a"}}}
	adapterProvider.UpdateOutbounds(nil, oldOpts)
	require.Len(t, adapterProvider.Outbounds(), 1)

	newOpts := []option.Outbound{{Tag: "node", Type: "test", Options: &testOutboundOptions{Marker: "b"}}}
	adapterProvider.UpdateOutbounds(oldOpts, newOpts)

	require.Len(t, adapterProvider.Outbounds(), 1)
	replaced, loaded := adapterProvider.Outbound("node")
	require.True(t, loaded)
	require.Equal(t, "b", replaced.(*testOutbound).marker)
	_, loaded = manager.Outbound("node[1]")
	require.False(t, loaded)
	require.Equal(t, []adapter.Outbound{replaced}, manager.Outbounds())
}
