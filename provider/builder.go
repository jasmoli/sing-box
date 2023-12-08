package provider

import (
	"context"
	"net/netip"
	"time"

	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/rw"
	"github.com/sagernet/sing/service/filemanager"
)

func New(ctx context.Context, router adapter.Router, logger log.ContextLogger, options option.OutboundProvider) (adapter.OutboundProvider, error) {
	if options.Path == "" {
		return nil, E.New("provider path missing")
	}
	path, _ := C.FindPath(options.Path)
	if foundPath, loaded := C.FindPath(path); loaded {
		path = foundPath
	}
	if !rw.FileExists(path) {
		path = filemanager.BasePath(ctx, path)
	}
	if options.HealthcheckUrl == "" {
		options.HealthcheckUrl = "https://www.gstatic.com/generate_204"
	}
	err := checkOverrideDialerOptions(options.OverrideDialer)
	if err != nil {
		return nil, E.Cause(err, `parse "override_dialer" failed: `)
	}
	switch options.Type {
	case C.TypeFileProvider:
		return NewFileProvider(ctx, router, logger, options, path)
	case C.TypeHTTPProvider:
		return NewHTTPProvider(ctx, router, logger, options, path)
	default:
		return nil, E.New("invalid provider type")
	}
}

func checkOverrideDialerOptions(dialerOptions map[string]any) error {
	for key, value := range dialerOptions {
		switch key {
		case "force_override":
			if _, exists := value.(bool); !exists {
				return E.New(`"force_override" must be boolean`)
			}
		case "detour":
			if _, exists := value.(string); !exists {
				return E.New(`"detour" must be string`)
			}
		case "bind_interface":
			if _, exists := value.(string); !exists {
				return E.New(`"bind_interface" must be string`)
			}
		case "inet4_bind_address":
			address, exists := value.(string)
			if !exists {
				return E.New(`"inet4_bind_address" must be string`)
			}
			_, err := netip.ParseAddr(address)
			if err != nil {
				return E.Cause(err, `invalid "inet4_bind_address"`)
			}
		case "inet6_bind_address":
			address, exists := value.(string)
			if !exists {
				return E.New(`"inet6_bind_address" must be string`)
			}
			_, err := netip.ParseAddr(address)
			if err != nil {
				return E.Cause(err, `invalid "inet6_bind_address"`)
			}
		case "routing_mark":
			if _, exists := value.(int); !exists {
				return E.New(`"routing_mark" must be int`)
			}
		case "reuse_addr":
			if _, exists := value.(bool); !exists {
				return E.New(`"reuse_addr" must be boolean`)
			}
		case "connect_timeout":
			duration, exists := value.(string)
			if !exists {
				return E.New(`"connect_timeout" must be string`)
			}
			_, err := time.ParseDuration(duration)
			if err != nil {
				return E.Cause(err, `invalid "connect_timeout"`)
			}
		case "tcp_fast_open":
			if _, exists := value.(bool); !exists {
				return E.New(`"tcp_fast_open" must be boolean`)
			}
		case "tcp_multi_path":
			if _, exists := value.(bool); !exists {
				return E.New(`"tcp_multi_path" must be boolean`)
			}
		case "udp_fragment":
			if _, exists := value.(bool); !exists {
				return E.New(`"udp_fragment" must be boolean`)
			}
		case "domain_strategy":
			strategy, exists := value.(string)
			if !exists {
				return E.New(`"domain_strategy" must be string`)
			}
			switch strategy {
			case "", "as_is":
			case "ipv4_only":
			case "ipv6_only":
			case "prefer_ipv4":
			case "prefer_ipv6":
			default:
				E.New(`unknown "domain_strategy" value: ` + strategy)
			}
		case "fallback_delay":
			duration, exists := value.(string)
			if !exists {
				return E.New(`"fallback_delay" must be string`)
			}
			_, err := time.ParseDuration(duration)
			if err != nil {
				return E.Cause(err, `invalid "fallback_delay"`)
			}
		default:
			return E.New(`unknown option: "` + key + `"`)
		}
	}
	return nil
}
