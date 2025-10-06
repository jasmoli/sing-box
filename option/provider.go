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
	Path                         string                                 `json:"path"`
	Exclude                      *badoption.Regexp                      `json:"exclude,omitempty"`
	Includes                     *badoption.Listable[*badoption.Regexp] `json:"include,omitempty"`
	Override                     *DialerOptions                         `json:"outbound_override"`
}

type ProviderLocalOptions struct {
	ProviderBaseOptions
	ProviderHealthCheckOptions
}

type ProviderRemoteOptions struct {
	ProviderBaseOptions
	ProviderHealthCheckOptions
	URL            string             `json:"download_url"`
	UserAgent      string             `json:"download_ua,omitempty"`
	DownloadDetour string             `json:"download_detour,omitempty"`
	UpdateInterval badoption.Duration `json:"download_interval,omitempty"`
}

type ProviderInlineOptions struct {
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
