package outbound

import (
	"github.com/sagernet/sing-box/option"
)

type Adapter struct {
	outboundType string
	outboundTag  string
	network      []string
	dependencies []string
	outboundPort uint16
}

func NewAdapter(outboundType string, outboundTag string, network []string, dependencies []string) Adapter {
	return Adapter{
		outboundType: outboundType,
		outboundTag:  outboundTag,
		network:      network,
		dependencies: dependencies,
	}
}

func NewAdapterWithDialerOptions(outboundType string, outboundTag string, network []string, dialOptions option.DialerOptions, serverOptions *option.ServerOptions) Adapter {
	var dependencies []string
	if dialOptions.Detour != "" {
		dependencies = []string{dialOptions.Detour}
	}
	adapter := NewAdapter(outboundType, outboundTag, network, dependencies)
	if serverOptions != nil {
		adapter.outboundPort = serverOptions.ServerPort
	}
	return adapter
}

func (a *Adapter) Type() string {
	return a.outboundType
}

func (a *Adapter) Tag() string {
	return a.outboundTag
}

func (a *Adapter) Port() uint16 {
	return a.outboundPort
}

func (a *Adapter) Network() []string {
	return a.network
}

func (a *Adapter) Dependencies() []string {
	return a.dependencies
}
