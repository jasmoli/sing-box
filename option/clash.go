package option

type ClashAPIOptions struct {
	ExternalController       string `json:"external_controller,omitempty"`
	ExternalUI               string `json:"external_ui,omitempty"`
	ExternalUIDownloadURL    string `json:"external_ui_download_url,omitempty"`
	ExternalUIDownloadDetour string `json:"external_ui_download_detour,omitempty"`
	Secret                   string `json:"secret,omitempty"`
	DefaultMode              string `json:"default_mode,omitempty"`
	StoreMode                bool   `json:"store_mode,omitempty"`
	StoreSelected            bool   `json:"store_selected,omitempty"`
	StoreFakeIP              bool   `json:"store_fakeip,omitempty"`
	CacheFile                string `json:"cache_file,omitempty"`
	CacheID                  string `json:"cache_id,omitempty"`

	ModeList []string `json:"-"`
}

type GroupOutboundOptions struct {
	Outbounds       Listable[string] `json:"outbounds,omitempty"`
	Providers       Listable[string] `json:"providers,omitempty"`
	UseAllProviders bool             `json:"use_all_providers,omitempty"`
	Includes        Listable[string] `json:"includes,omitempty"`
	Excludes        string           `json:"excludes,omitempty"`
	Types           Listable[string] `json:"types,omitempty"`
	Ports           Listable[string] `json:"ports,omitempty"`
}

type SelectorOutboundOptions struct {
	GroupOutboundOptions
	Default                   string   `json:"default,omitempty"`
	InterruptExistConnections bool     `json:"interrupt_exist_connections,omitempty"`
}

type URLTestOutboundOptions struct {
	GroupOutboundOptions
	URL                       string   `json:"url,omitempty"`
	Interval                  Duration `json:"interval,omitempty"`
	Tolerance                 uint16   `json:"tolerance,omitempty"`
	InterruptExistConnections bool     `json:"interrupt_exist_connections,omitempty"`
}
