package option

import "github.com/sagernet/sing/common/json/badoption"

type FilterOptions struct {
	Exclude  *badoption.Regexp                      `json:"exclude,omitempty"`
	Includes *badoption.Listable[*badoption.Regexp] `json:"include,omitempty"`
	Types    *badoption.Listable[string]            `json:"types,omitempty"`
	Ports    *badoption.Listable[string]            `json:"ports,omitempty"`
}

type SelectorOutboundOptions struct {
	GroupCommonOption
	Default                   string `json:"default,omitempty"`
	InterruptExistConnections bool   `json:"interrupt_exist_connections,omitempty"`
}

type URLTestOutboundOptions struct {
	GroupCommonOption
	URL                       string             `json:"url,omitempty"`
	Interval                  badoption.Duration `json:"interval,omitempty"`
	Tolerance                 uint16             `json:"tolerance,omitempty"`
	IdleTimeout               badoption.Duration `json:"idle_timeout,omitempty"`
	InterruptExistConnections bool               `json:"interrupt_exist_connections,omitempty"`
}

type GroupCommonOption struct {
	FilterOptions
	Outbounds       []string          `json:"outbounds"`
	Providers       []string          `json:"providers"`
	UseAllProviders bool              `json:"use_all_providers,omitempty"`
}
