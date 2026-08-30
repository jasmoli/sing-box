//go:build with_ebpf && (linux || android)

package ebpf

// Lifecycle for the legacy cgroup data plane.

import (
	"github.com/sagernet/sing-box/adapter"
	commonEBPF "github.com/sagernet/sing-box/common/ebpf"
	E "github.com/sagernet/sing/common/exceptions"
)

func (i *Inbound) startCgroupInbound(stage adapter.StartStage) error {
	switch stage {
	case adapter.StartStateInitialize:
		if len(i.localBypassPort) > 0 || len(i.sharedBypassPort) > 0 {
			i.logger.Warn("bypass_port is only implemented by the TC data plane and is ignored by the legacy cgroup data plane")
		}
		if err := i.startSocketProtection(); err != nil {
			return err
		}
		if err := i.selectRedirectPrefixes(); err != nil {
			return err
		}
		if i.localEnabled && i.androidUIDOptions == nil {
			if err := i.prepareCgroupBackend(); err != nil {
				return err
			}
		}
		if i.sharedEnabled {
			i.sharedNetwork = newSharedNetwork(i, i.sharedOptions)
		}
	case adapter.StartStateStart:
		if i.localEnabled && i.androidUIDOptions != nil {
			if err := i.resolveAndroidUIDPolicy(); err != nil {
				return combineStartError(E.Cause(err, "resolve Android UID policy"), i.cleanupStartFailure())
			}
			if err := i.prepareCgroupBackend(); err != nil {
				return combineStartError(err, i.cleanupStartFailure())
			}
		}
		backend := i.cgroupBackendInstance()
		if i.localEnabled && backend == nil {
			return combineStartError(E.New("eBPF backend is not initialized"), i.cleanupStartFailure())
		}
		if err := i.startBypassRuleSets(); err != nil {
			return combineStartError(
				E.Cause(err, "initialize eBPF bypass_rule_set"),
				i.cleanupStartFailure(),
			)
		}
		if err := i.setupLocalRoutes(); err != nil {
			return combineStartError(
				E.Cause(err, "configure eBPF redirect routes"),
				i.cleanupStartFailure(),
			)
		}
		if i.localEnabled {
			if err := i.startListeners(); err != nil {
				return combineStartError(err, i.cleanupStartFailure())
			}
			if err := backend.LoadPrograms(i.listeners.selectedPort()); err != nil {
				return combineStartError(err, i.cleanupStartFailure())
			}
		}
		if i.sharedNetwork != nil {
			if err := i.sharedNetwork.Start(backend); err != nil {
				return combineStartError(err, i.cleanupStartFailure())
			}
		}
		if i.localEnabled {
			if err := backend.Attach(); err != nil {
				return combineStartError(err, i.cleanupStartFailure())
			}
			if i.enableTCP {
				i.startTCPRedirectJanitor()
			}
			bypassIPv4Count, bypassIPv6Count := backend.BypassCIDRCount()
			i.logger.Info(
				"eBPF local cgroup interception ready: cgroup=", backend.CgroupPath(),
				", redirect_listener_port=", i.listeners.selectedPort(),
				", dns_mode=", i.localDNSMode,
				", ipv6=", i.cgroupIPv6Enabled(),
				", bypass_private_address=", i.localPolicy.BypassPrivateAddress,
				", udp_state_cleanup=", backend.UDPCleanupMode(),
				", uid_policy={include_configured:", i.localPolicy.IncludeUIDConfigured,
				", include:[", formatUIDRanges(i.localPolicy.IncludeUID), "]",
				", exclude:[", formatUIDRanges(i.localPolicy.ExcludeUID), "]}",
				", bypass_cidr={ipv4:", bypassIPv4Count, ", ipv6:", bypassIPv6Count, "}",
			)
		}
	}
	return nil
}

func (i *Inbound) prepareCgroupBackend() error {
	policy := i.localPolicy
	policy.EnableBypassCIDR = true
	backend, err := commonEBPF.PrepareCgroup(commonEBPF.CgroupConfig{
		Path:         i.cgroupPath,
		EnableTCP:    i.enableTCP,
		EnableUDP:    i.enableUDP,
		EnableIPv6:   i.cgroupIPv6Enabled(),
		RedirectIPv4: i.redirectIPv4Prefix,
		RedirectIPv6: i.redirectIPv6Prefix,
		FakeIPIPv4:   i.fakeIPIPv4Prefix,
		FakeIPIPv6:   i.fakeIPIPv6Prefix,
		MapCapacity:  i.cgroupMapCapacity,
		UDPTimeout:   i.udpTimeout,
		Policy:       policy,
	})
	if err != nil {
		return err
	}
	if err = i.socketProtector.Attach(backend); err != nil {
		closeErr := backend.Close()
		if closeErr != nil {
			closeErr = E.Cause(closeErr, "close eBPF backend")
		}
		return E.Errors(err, closeErr)
	}
	i.setCgroupBackend(backend)
	return nil
}

func (i *Inbound) startSocketProtection() error {
	if i.socketProtector == nil || i.socketProtectionRegistration != nil {
		return nil
	}
	registration, err := adapter.RegisterEBPFSocketProtection(i.ctx, i.socketProtector.ControlFunc())
	if err != nil {
		return E.Cause(err, "register eBPF socket protection")
	}
	i.socketProtectionRegistration = registration
	return nil
}

func (i *Inbound) closeCgroupResources() error {
	i.stopTCPRedirectJanitor()
	i.stopBypassRuleSets()
	i.closeSocketProtection()
	backend := i.takeCgroupBackend()
	var sharedErr error
	if i.sharedNetwork != nil {
		sharedErr = i.sharedNetwork.Close()
		if !i.sharedNetwork.IsClosed() {
			i.setCgroupBackend(backend)
			if sharedErr == nil {
				sharedErr = E.New("shared-network eBPF backend remained open after close")
			}
			return sharedErr
		}
		i.sharedNetwork = nil
	}
	var backendErr error
	if backend != nil {
		backendErr = backend.Close()
		if !backend.IsClosed() {
			i.setCgroupBackend(backend)
			if backendErr == nil {
				backendErr = E.New("eBPF backend remained open after close")
			}
			return backendErr
		}
	}
	listenerErr := i.closeListeners()
	i.udpNat.Purge()
	return E.Errors(sharedErr, backendErr, listenerErr, i.removeLocalRoutes())
}

func (i *Inbound) cgroupBackendInstance() *commonEBPF.CgroupBackend {
	i.cgroupBackendAccess.RLock()
	defer i.cgroupBackendAccess.RUnlock()
	return i.cgroupBackend
}

func (i *Inbound) setCgroupBackend(backend *commonEBPF.CgroupBackend) {
	i.cgroupBackendAccess.Lock()
	i.cgroupBackend = backend
	i.cgroupBackendAccess.Unlock()
}

func (i *Inbound) takeCgroupBackend() *commonEBPF.CgroupBackend {
	i.cgroupBackendAccess.Lock()
	backend := i.cgroupBackend
	i.cgroupBackend = nil
	i.cgroupBackendAccess.Unlock()
	return backend
}

func (i *Inbound) redirectAddressStrings() []string {
	addresses := make([]string, 0, 2)
	if i.redirectIPv4Prefix.IsValid() {
		addresses = append(addresses, i.redirectIPv4Prefix.String())
	}
	if i.redirectIPv6Prefix.IsValid() {
		addresses = append(addresses, i.redirectIPv6Prefix.String())
	}
	return addresses
}

func (i *Inbound) closeSocketProtection() {
	if i.socketProtectionRegistration != nil {
		i.socketProtectionRegistration.Close()
		i.socketProtectionRegistration = nil
	}
	if i.socketProtector != nil {
		i.socketProtector.Close()
	}
}
