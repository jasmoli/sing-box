package option

import (
	"context"

	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/json"
	"github.com/sagernet/sing/common/json/badjson"
	"github.com/sagernet/sing/common/json/badoption"
	"github.com/sagernet/sing/service"
)

type ProviderOptionsRegistry interface {
	CreateOptions(providerType string) (any, bool)
}
type _Provider struct {
	Type    string `json:"type"`
	Tag     string `json:"tag,omitempty"`
	Options any    `json:"-"`
}

type Provider _Provider

func (h *Provider) MarshalJSONContext(ctx context.Context) ([]byte, error) {
	return badjson.MarshallObjectsContext(ctx, (*_Provider)(h), h.Options)
}

func (h *Provider) UnmarshalJSONContext(ctx context.Context, content []byte) error {
	err := json.UnmarshalContext(ctx, content, (*_Provider)(h))
	if err != nil {
		return err
	}
	registry := service.FromContext[ProviderOptionsRegistry](ctx)
	if registry == nil {
		return E.New("missing provider options registry in context")
	}
	options, loaded := registry.CreateOptions(h.Type)
	if !loaded {
		return E.New("unknown provider type: ", h.Type)
	}
	err = badjson.UnmarshallExcludedContext(ctx, content, (*_Provider)(h), options)
	if err != nil {
		return err
	}
	h.Options = options
	return nil
}

type ProviderBaseOptions struct {
	Path     string                   `json:"path"`
	Override *ProviderOverrideOptions `json:"outbound_override"`
}

type ProviderOverrideOptions struct {
	TagPrefix string `json:"tag_prefix,omitempty"`
	TagSuffix string `json:"tag_suffix,omitempty"`
	*OverrideDialerOptions
}

type OverrideDialerOptions struct {
	Detour              *string                            `json:"detour,omitempty"`
	BindInterface       *string                            `json:"bind_interface,omitempty"`
	Inet4BindAddress    *badoption.Addr                    `json:"inet4_bind_address,omitempty"`
	Inet6BindAddress    *badoption.Addr                    `json:"inet6_bind_address,omitempty"`
	ProtectPath         *string                            `json:"protect_path,omitempty"`
	RoutingMark         *FwMark                            `json:"routing_mark,omitempty"`
	ReuseAddr           *bool                              `json:"reuse_addr,omitempty"`
	NetNs               *string                            `json:"netns,omitempty"`
	ConnectTimeout      *badoption.Duration                `json:"connect_timeout,omitempty"`
	TCPFastOpen         *bool                              `json:"tcp_fast_open,omitempty"`
	TCPMultiPath        *bool                              `json:"tcp_multi_path,omitempty"`
	UDPFragment         *bool                              `json:"udp_fragment,omitempty"`
	UDPFragmentDefault  *bool                              `json:"-"`
	DomainResolver      *DomainResolveOptions              `json:"domain_resolver,omitempty"`
	NetworkStrategy     *NetworkStrategy                   `json:"network_strategy,omitempty"`
	NetworkType         *badoption.Listable[InterfaceType] `json:"network_type,omitempty"`
	FallbackNetworkType *badoption.Listable[InterfaceType] `json:"fallback_network_type,omitempty"`
	FallbackDelay       *badoption.Duration                `json:"fallback_delay,omitempty"`

	// Deprecated: migrated to domain resolver
	DomainStrategy *DomainStrategy `json:"domain_strategy,omitempty"`
}

type ProviderLocalOptions struct {
	ProviderBaseOptions
	ProviderHealthCheckOptions
}

type ProviderRemoteOptions struct {
	FilterOptions
	ProviderBaseOptions
	ProviderHealthCheckOptions
	URL            string             `json:"download_url"`
	Path           string             `json:"path,omitempty"`
	UserAgent      string             `json:"download_ua,omitempty"`
	DownloadDetour string             `json:"download_detour,omitempty"`
	UpdateInterval badoption.Duration `json:"download_interval,omitempty"`
}

type ProviderInlineOptions struct {
	ProviderBaseOptions
	ProviderHealthCheckOptions
	Outbounds   []Outbound                 `json:"outbounds,omitempty"`
}

type ProviderHealthCheckOptions struct {
	EnabledHealthCheck           bool               `json:"enable_healthcheck"`
	HealthCheckURL               string             `json:"healthcheck_url"`
	HealthCheckInterval          badoption.Duration `json:"healthcheck_interval,omitempty"`
	HealthCheckWhenNetworkChange bool               `json:"healthcheck_when_network_change,omitempty"`
	HealthCheckTimeout           badoption.Duration `json:"healthcheck_timeout"`
}
