package parser

import (
	"context"
	"reflect"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
	E "github.com/sagernet/sing/common/exceptions"
)

var subscriptionParsers = []func(ctx context.Context, content string) ([]option.Outbound, error){
	ParseBoxSubscription,
	ParseClashSubscription,
	ParseSIP008Subscription,
	ParseRawSubscription,
}

func ParseSubscription(ctx context.Context, content string, override *option.ProviderOverrideOptions) ([]option.Outbound, error) {
	var pErr error
	for _, parser := range subscriptionParsers {
		servers, err := parser(ctx, content)
		if len(servers) > 0 {
			if override != nil {
				servers = overrideOutbounds(servers, override)
			}
			return servers, nil
		}
		pErr = E.Errors(pErr, err)
	}
	return nil, E.Cause(pErr, "no servers found")
}

func overrideOutbounds(outbounds []option.Outbound, override *option.ProviderOverrideOptions) []option.Outbound {
	var tags []string
	for _, outbound := range outbounds {
		tags = append(tags, outbound.Tag)
	}
	var parsedOutbounds []option.Outbound
	for _, outbound := range outbounds {
		if override != nil {
			if override.TagPrefix != "" {
				outbound.Tag = override.TagPrefix + outbound.Tag
			}
			if override.TagSuffix != "" {
				outbound.Tag = outbound.Tag + override.TagSuffix
			}
		}
		switch outbound.Type {
		case C.TypeHTTP:
			options := outbound.Options.(*option.HTTPOutboundOptions)
			options.DialerOptions = overrideDialerOption(options.DialerOptions, tags, override)
			outbound.Options = options
		case C.TypeSOCKS:
			options := outbound.Options.(*option.SOCKSOutboundOptions) // 注意：应该是 SOCKS 不是 Socks
			options.DialerOptions = overrideDialerOption(options.DialerOptions, tags, override)
			outbound.Options = options
		case C.TypeTUIC:
			options := outbound.Options.(*option.TUICOutboundOptions)
			options.DialerOptions = overrideDialerOption(options.DialerOptions, tags, override)
			outbound.Options = options
		case C.TypeVMess:
			options := outbound.Options.(*option.VMessOutboundOptions)
			options.DialerOptions = overrideDialerOption(options.DialerOptions, tags, override)
			outbound.Options = options
		case C.TypeVLESS:
			options := outbound.Options.(*option.VLESSOutboundOptions)
			options.DialerOptions = overrideDialerOption(options.DialerOptions, tags, override)
			outbound.Options = options
		case C.TypeTrojan:
			options := outbound.Options.(*option.TrojanOutboundOptions)
			options.DialerOptions = overrideDialerOption(options.DialerOptions, tags, override)
			outbound.Options = options
		case C.TypeHysteria:
			options := outbound.Options.(*option.HysteriaOutboundOptions)
			options.DialerOptions = overrideDialerOption(options.DialerOptions, tags, override)
			outbound.Options = options
		case C.TypeShadowTLS:
			options := outbound.Options.(*option.ShadowTLSOutboundOptions)
			options.DialerOptions = overrideDialerOption(options.DialerOptions, tags, override)
			outbound.Options = options
		case C.TypeHysteria2:
			options := outbound.Options.(*option.Hysteria2OutboundOptions)
			options.DialerOptions = overrideDialerOption(options.DialerOptions, tags, override)
			outbound.Options = options
		case C.TypeWireGuard:
			options := outbound.Options.(*option.WireGuardEndpointOptions)
			options.DialerOptions = overrideDialerOption(options.DialerOptions, tags, override)
			outbound.Options = options
		case C.TypeShadowsocks:
			options := outbound.Options.(*option.ShadowsocksOutboundOptions)
			options.DialerOptions = overrideDialerOption(options.DialerOptions, tags, override)
			outbound.Options = options
		case C.TypeAnyTLS:
			options := outbound.Options.(*option.AnyTLSOutboundOptions)
			options.DialerOptions = overrideDialerOption(options.DialerOptions, tags, override)
			outbound.Options = options
		}
		// dialer := outbound.Options.(option.DialerOptions)
		// overrideDialerOption(dialer, tags, override)
		parsedOutbounds = append(parsedOutbounds, outbound)
	}
	return parsedOutbounds
}

func overrideDialerOption(options option.DialerOptions, tags []string, override *option.ProviderOverrideOptions) option.DialerOptions {
	if options.Detour != "" && !common.Any(tags, func(tag string) bool {
		return options.Detour == tag
	}) {
		options.Detour = ""
	}
	var defaultOptions option.DialerOptions
	if override == nil || override.OverrideDialerOptions == nil || reflect.DeepEqual(override.OverrideDialerOptions, defaultOptions) {
		return options
	}
	if override.OverrideDialerOptions.Detour != nil && options.Detour == "" {
		options.Detour = *override.OverrideDialerOptions.Detour
	}
	if override.OverrideDialerOptions.BindInterface != nil {
		options.BindInterface = *override.OverrideDialerOptions.BindInterface
	}
	if override.OverrideDialerOptions.Inet4BindAddress != nil {
		options.Inet4BindAddress = override.OverrideDialerOptions.Inet4BindAddress
	}
	if override.OverrideDialerOptions.Inet6BindAddress != nil {
		options.Inet6BindAddress = override.OverrideDialerOptions.Inet6BindAddress
	}
	if override.OverrideDialerOptions.ProtectPath != nil {
		options.ProtectPath = *override.OverrideDialerOptions.ProtectPath
	}
	if override.OverrideDialerOptions.RoutingMark != nil {
		options.RoutingMark = *override.OverrideDialerOptions.RoutingMark
	}
	if override.OverrideDialerOptions.ReuseAddr != nil {
		options.ReuseAddr = *override.OverrideDialerOptions.ReuseAddr
	}
	if override.OverrideDialerOptions.ConnectTimeout != nil {
		options.ConnectTimeout = *override.OverrideDialerOptions.ConnectTimeout
	}
	if override.OverrideDialerOptions.TCPFastOpen != nil {
		options.TCPFastOpen = *override.OverrideDialerOptions.TCPFastOpen
	}
	if override.OverrideDialerOptions.TCPMultiPath != nil {
		options.TCPMultiPath = *override.OverrideDialerOptions.TCPMultiPath
	}
	if override.OverrideDialerOptions.UDPFragment != nil {
		options.UDPFragment = override.OverrideDialerOptions.UDPFragment
	}
	if override.OverrideDialerOptions.UDPFragmentDefault != nil {
		options.UDPFragmentDefault = *override.OverrideDialerOptions.UDPFragmentDefault
	}
	if override.OverrideDialerOptions.DomainResolver != nil {
		options.DomainResolver = override.OverrideDialerOptions.DomainResolver
	}
	if override.OverrideDialerOptions.NetworkStrategy != nil {
		options.NetworkStrategy = override.OverrideDialerOptions.NetworkStrategy
	}
	if override.OverrideDialerOptions.NetworkType != nil {
		options.NetworkType = *override.OverrideDialerOptions.NetworkType
	}
	if override.OverrideDialerOptions.FallbackNetworkType != nil {
		options.FallbackNetworkType = *override.OverrideDialerOptions.FallbackNetworkType
	}
	if override.OverrideDialerOptions.FallbackDelay != nil {
		options.FallbackDelay = *override.OverrideDialerOptions.FallbackDelay
	}

	// Deprecated: migrated to domain resolver
	if override.OverrideDialerOptions.DomainStrategy != nil {
		options.UDPFragment = override.OverrideDialerOptions.UDPFragment
	}
	return options
}
