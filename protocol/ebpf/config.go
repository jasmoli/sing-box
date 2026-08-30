//go:build with_ebpf && (linux || android)

package ebpf

import (
	"net"
	"net/netip"
	"strconv"
	"strings"

	commonEBPF "github.com/sagernet/sing-box/common/ebpf"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/json/badoption"
)

func normalizeMode(mode string) (string, bool, bool, error) {
	switch mode {
	case "", ebpfModeLocal:
		return ebpfModeLocal, true, false, nil
	case ebpfModeShared:
		return ebpfModeShared, false, true, nil
	case ebpfModeHybrid:
		return ebpfModeHybrid, true, true, nil
	default:
		return "", false, false, E.New("unknown eBPF mode: ", mode)
	}
}

func validateLocalOptions(enabled bool, options option.EBPFLocalOptions) error {
	if enabled {
		return nil
	}
	if options.DNSMode != "" {
		return E.New("local.dns_mode requires local or hybrid mode")
	}
	if options.IPv6 != nil {
		return E.New("local.ipv6 requires local or hybrid mode")
	}
	if options.BypassPrivateAddress != nil {
		return E.New("local.bypass_private_address requires local or hybrid mode")
	}
	if len(options.IncludeUID) > 0 || len(options.IncludeUIDRange) > 0 ||
		len(options.ExcludeUID) > 0 || len(options.ExcludeUIDRange) > 0 ||
		len(options.IncludeAndroidUser) > 0 || len(options.IncludePackage) > 0 ||
		len(options.ExcludePackage) > 0 {
		return E.New("local UID policy requires local or hybrid mode")
	}
	return nil
}

func validateAndroidUIDOptions(goos string, options option.EBPFLocalOptions) error {
	if !hasAndroidUIDOptions(options) {
		return nil
	}
	if goos != "android" {
		return E.New("include_android_user, include_package, and exclude_package are only supported on Android")
	}
	const maxAndroidUserID = (uint64(^uint32(0)-1) - (androidUserRange - 1)) / androidUserRange
	for _, userID := range options.IncludeAndroidUser {
		if userID < 0 || uint64(userID) > maxAndroidUserID {
			return E.New("invalid include_android_user: ", userID)
		}
	}
	return nil
}

func hasAndroidUIDOptions(options option.EBPFLocalOptions) bool {
	return len(options.IncludeAndroidUser) > 0 || len(options.IncludePackage) > 0 || len(options.ExcludePackage) > 0
}

func normalizeDNSMode(mode string) (string, error) {
	switch mode {
	case "", dnsModeRespectPolicy:
		return dnsModeRespectPolicy, nil
	case dnsModeHijack, dnsModeOff:
		return mode, nil
	default:
		return "", E.New("unknown eBPF dns_mode: ", mode)
	}
}

func enabledByDefault(value *bool) bool {
	return value == nil || *value
}

func parseUIDRanges(uidList []uint32, rangeList []string) ([]commonEBPF.UIDRange, error) {
	uidRanges := make([]commonEBPF.UIDRange, 0, len(uidList)+len(rangeList))
	for _, uid := range uidList {
		uidRanges = append(uidRanges, commonEBPF.UIDRange{Start: uid, End: uid})
	}
	for _, uidRange := range rangeList {
		separator := strings.IndexByte(uidRange, ':')
		if separator < 0 {
			return nil, E.New("missing ':' in range: ", uidRange)
		}
		if separator == 0 {
			return nil, E.New("missing range start: ", uidRange)
		}
		if separator == len(uidRange)-1 {
			return nil, E.New("missing range end: ", uidRange)
		}
		start, err := strconv.ParseUint(uidRange[:separator], 0, 32)
		if err != nil {
			return nil, E.Cause(err, "parse range start")
		}
		end, err := strconv.ParseUint(uidRange[separator+1:], 0, 32)
		if err != nil {
			return nil, E.Cause(err, "parse range end")
		}
		if start > end {
			return nil, E.New("range start is greater than range end: ", uidRange)
		}
		uidRanges = append(uidRanges, commonEBPF.UIDRange{Start: uint32(start), End: uint32(end)})
	}
	return uidRanges, nil
}

func validateSharedOptions(enabled bool, options option.EBPFSharedOptions) error {
	if enabled {
		return nil
	}
	if options.DNSMode != "" || len(options.Interface) > 0 || options.IPv6 != nil || options.BypassPrivateAddress != nil ||
		len(options.IncludeSourceCIDR) > 0 || len(options.ExcludeSourceCIDR) > 0 ||
		len(options.IncludeMACAddress) > 0 || len(options.ExcludeMACAddress) > 0 {
		return E.New("shared options require shared or hybrid mode")
	}
	return nil
}

func normalizeSharedOptions(options option.EBPFSharedOptions) (option.EBPFSharedOptions, error) {
	if len(options.Interface) == 0 {
		return option.EBPFSharedOptions{}, E.New("shared.interface must not be empty")
	}
	seen := make(map[string]struct{}, len(options.Interface))
	interfaces := make(badoption.Listable[string], 0, len(options.Interface))
	for _, interfaceName := range options.Interface {
		interfaceName = strings.TrimSpace(interfaceName)
		if interfaceName == "" {
			return option.EBPFSharedOptions{}, E.New("shared.interface contains an empty interface name")
		}
		if interfaceName == "lo" {
			return option.EBPFSharedOptions{}, E.New("shared.interface must not contain lo")
		}
		if _, loaded := seen[interfaceName]; loaded {
			continue
		}
		seen[interfaceName] = struct{}{}
		interfaces = append(interfaces, interfaceName)
	}
	options.Interface = interfaces
	var err error
	options.IncludeSourceCIDR, err = normalizeSourceCIDR("include_source_cidr", options.IncludeSourceCIDR)
	if err != nil {
		return option.EBPFSharedOptions{}, err
	}
	options.ExcludeSourceCIDR, err = normalizeSourceCIDR("exclude_source_cidr", options.ExcludeSourceCIDR)
	if err != nil {
		return option.EBPFSharedOptions{}, err
	}
	return options, nil
}

func normalizeSourceCIDR(name string, prefixes []netip.Prefix) (badoption.Listable[netip.Prefix], error) {
	normalized := make(badoption.Listable[netip.Prefix], 0, len(prefixes))
	seen := make(map[netip.Prefix]struct{}, len(prefixes))
	for _, prefix := range prefixes {
		if !prefix.IsValid() {
			return nil, E.New("invalid shared.", name)
		}
		prefix = prefix.Masked()
		if prefix.Addr().Is4In6() && prefix.Bits() >= 96 {
			prefix = netip.PrefixFrom(prefix.Addr().Unmap(), prefix.Bits()-96).Masked()
		}
		if _, loaded := seen[prefix]; loaded {
			continue
		}
		seen[prefix] = struct{}{}
		normalized = append(normalized, prefix)
	}
	return normalized, nil
}

func parseSharedMACAddresses(name string, addresses []string) ([]commonEBPF.MACAddress, error) {
	parsed := make([]commonEBPF.MACAddress, 0, len(addresses))
	seen := make(map[commonEBPF.MACAddress]struct{}, len(addresses))
	for index, address := range addresses {
		hardwareAddress, err := net.ParseMAC(address)
		if err != nil {
			return nil, E.Cause(err, "parse shared.", name, "[", index, "]")
		}
		if len(hardwareAddress) != len(commonEBPF.MACAddress{}) {
			return nil, E.New("shared.", name, "[", index, "] must be a 48-bit MAC address")
		}
		var mac commonEBPF.MACAddress
		copy(mac[:], hardwareAddress)
		if _, loaded := seen[mac]; loaded {
			continue
		}
		seen[mac] = struct{}{}
		parsed = append(parsed, mac)
	}
	return parsed, nil
}
