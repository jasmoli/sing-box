package provider

import (
	"net/netip"
	"runtime"
	"strings"
	"time"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-dns"
	E "github.com/sagernet/sing/common/exceptions"
)

func newParser(content string, dialerOptions map[string]any) ([]option.Outbound, error) {
	var outbounds []option.Outbound
	var err error
	if strings.Contains(content, "\"outbounds\"") {
		var options option.Options
		err = options.UnmarshalJSON([]byte(content))
		if err != nil {
			return nil, E.Cause(err, "decode config at ")
		}
		outbounds = options.Outbounds
		return overrideOutbounds(outbounds, dialerOptions), nil
	} else if strings.Contains(content, "proxies") {
		outbounds, err = newClashParser(content)
		if err != nil {
			return nil, err
		}
		return overrideOutbounds(outbounds, dialerOptions), nil
	}
	outbounds, err = newNativeURIParser(content)
	if err != nil {
		return nil, err
	}
	return overrideOutbounds(outbounds, dialerOptions), err
}

func overrideOutbounds(outbounds []option.Outbound, dialerOptions map[string]any) []option.Outbound {
	parsedOutbounds := []option.Outbound{}
	for _, outbound := range outbounds {
		switch outbound.Type {
		case C.TypeHTTP:
			dialer := outbound.HTTPOptions.DialerOptions
			outbound.HTTPOptions.DialerOptions = overrideDialerOption(dialer, dialerOptions)
		case C.TypeSOCKS:
			dialer := outbound.SocksOptions.DialerOptions
			outbound.SocksOptions.DialerOptions = overrideDialerOption(dialer, dialerOptions)
		case C.TypeTUIC:
			dialer := outbound.TUICOptions.DialerOptions
			outbound.TUICOptions.DialerOptions = overrideDialerOption(dialer, dialerOptions)
		case C.TypeVMess:
			dialer := outbound.VMessOptions.DialerOptions
			outbound.VMessOptions.DialerOptions = overrideDialerOption(dialer, dialerOptions)
		case C.TypeVLESS:
			dialer := outbound.VLESSOptions.DialerOptions
			outbound.VLESSOptions.DialerOptions = overrideDialerOption(dialer, dialerOptions)
		case C.TypeTrojan:
			dialer := outbound.TrojanOptions.DialerOptions
			outbound.TrojanOptions.DialerOptions = overrideDialerOption(dialer, dialerOptions)
		case C.TypeHysteria:
			dialer := outbound.HysteriaOptions.DialerOptions
			outbound.HysteriaOptions.DialerOptions = overrideDialerOption(dialer, dialerOptions)
		case C.TypeShadowTLS:
			dialer := outbound.ShadowTLSOptions.DialerOptions
			outbound.ShadowTLSOptions.DialerOptions = overrideDialerOption(dialer, dialerOptions)
		case C.TypeHysteria2:
			dialer := outbound.Hysteria2Options.DialerOptions
			outbound.Hysteria2Options.DialerOptions = overrideDialerOption(dialer, dialerOptions)
		case C.TypeWireGuard:
			dialer := outbound.WireGuardOptions.DialerOptions
			outbound.WireGuardOptions.DialerOptions = overrideDialerOption(dialer, dialerOptions)
		case C.TypeShadowsocks:
			dialer := outbound.ShadowsocksOptions.DialerOptions
			outbound.ShadowsocksOptions.DialerOptions = overrideDialerOption(dialer, dialerOptions)
		case C.TypeShadowsocksR:
			dialer := outbound.ShadowsocksROptions.DialerOptions
			outbound.ShadowsocksROptions.DialerOptions = overrideDialerOption(dialer, dialerOptions)
		}
		parsedOutbounds = append(parsedOutbounds, outbound)
	}
	return parsedOutbounds
}

func overrideDialerOption(options option.DialerOptions, dialerOptions map[string]any) option.DialerOptions {
	if len(dialerOptions) == 0 {
		return options
	}
	force := false
	if forceOverride, ok := dialerOptions["force_override"].(bool); ok && forceOverride {
		force = true
	}
	for key, value := range dialerOptions {
		switch key {
		case "detour":
			if options.Detour != "" && !force {
				continue
			}
			options.Detour = value.(string)
		case "bind_interface":
			if options.BindInterface != "" && !force {
				continue
			}
			options.BindInterface = value.(string)
		case "inet4_bind_address":
			if options.Inet4BindAddress != nil && !force {
				continue
			}
			addr, _ := netip.ParseAddr(value.(string))
			options.Inet4BindAddress = option.NewListenAddress(addr)
		case "inet6_bind_address":
			if options.Inet6BindAddress != nil && !force {
				continue
			}
			addr, _ := netip.ParseAddr(value.(string))
			options.Inet6BindAddress = option.NewListenAddress(addr)
		case "routing_mark":
			if runtime.GOOS != "android" && runtime.GOOS != "linux" {
				continue
			}
			if options.RoutingMark != 0 && !force {
				continue
			}
			options.RoutingMark = value.(int)
		case "reuse_addr":
			options.ReuseAddr = value.(bool)
		case "connect_timeout":
			if options.ConnectTimeout != 0 && !force {
				continue
			}
			duration, _ := time.ParseDuration(value.(string))
			options.ConnectTimeout = option.Duration(duration)
		case "tcp_fast_open":
			options.TCPFastOpen = value.(bool)
		case "tcp_multi_path":
			options.TCPMultiPath = value.(bool)
		case "udp_fragment":
			parsedValue := value.(bool)
			options.UDPFragment = &parsedValue
		case "domain_strategy":
			asis := option.DomainStrategy(dns.DomainStrategyAsIS)
			if options.DomainStrategy != asis && !force {
				continue
			}
			var strategy option.DomainStrategy
			switch value.(string) {
			case "", "as_is":
				strategy = asis
			case "ipv4_only":
				strategy = option.DomainStrategy(dns.DomainStrategyUseIPv4)
			case "ipv6_only":
				strategy = option.DomainStrategy(dns.DomainStrategyUseIPv6)
			case "prefer_ipv4":
				strategy = option.DomainStrategy(dns.DomainStrategyPreferIPv4)
			case "prefer_ipv6":
				strategy = option.DomainStrategy(dns.DomainStrategyPreferIPv4)
			}
			options.DomainStrategy = strategy
		case "fallback_delay":
			if options.FallbackDelay != 0 && !force {
				continue
			}
			delay, _ := time.ParseDuration(value.(string))
			options.FallbackDelay = option.Duration(delay)
		}
	}
	return options
}
