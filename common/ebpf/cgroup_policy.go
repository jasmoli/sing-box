//go:build with_ebpf && (linux || android)

package ebpf

// CgroupPolicy is the legacy cgroup data plane policy. It carries the same
// fields as LocalPolicy, so it is an alias: both data planes compile identical
// UID and bypass-CIDR policy maps.
type CgroupPolicy = LocalPolicy
