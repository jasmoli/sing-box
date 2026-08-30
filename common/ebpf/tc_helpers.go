//go:build with_ebpf && (linux || android)

package ebpf

import (
	"net/netip"

	E "github.com/sagernet/sing/common/exceptions"
)

func normalizeAddressPrefix(name string, prefix netip.Prefix, ipv4 bool) (netip.Prefix, error) {
	if !prefix.IsValid() {
		return netip.Prefix{}, nil
	}
	prefix = prefix.Masked()
	if prefix.Addr().Is4() != ipv4 || prefix.Addr().Is4In6() {
		return netip.Prefix{}, E.New("invalid ", name, ": ", prefix)
	}
	return prefix, nil
}
