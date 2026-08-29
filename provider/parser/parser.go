package parser

import (
	"context"
	"reflect"
	"strings"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
)

func ParseSubscription(ctx context.Context, content string, override *option.ProviderOverrideOptions) ([]option.Outbound, error) {
	var outbounds []option.Outbound
	var err error
	if strings.Contains(content, "\"outbounds\"") {
		outbounds, err = ParseBoxSubscription(ctx, content)
	} else if strings.Contains(content, "proxies") {
		outbounds, err = ParseClashSubscription(ctx, content)
	} else {
		outbounds, err = ParseRawSubscription(ctx, content)
	}
	if err != nil {
		return nil, err
	}
	if override != nil {
		outbounds = overrideOutbounds(outbounds, override)
	}
	return outbounds, nil
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
			options := outbound.Options.(*option.SOCKSOutboundOptions)
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
	return options
}
