//go:build with_ebpf && (linux || android)

package ebpf

// Client UDP state for the legacy cgroup data plane. The TC data plane keeps its
// own state in udp_state.go; only one data plane is active per process.

import (
	"net"
	"net/netip"
	"sync"
	"sync/atomic"

	commonEBPF "github.com/sagernet/sing-box/common/ebpf"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type cgroupUDPClientTable struct {
	clientShards       [cgroupUDPClientShardCount]cgroupUDPClientShard
	redirectAccess     sync.Mutex
	redirectReferences map[udpRedirectReference]uint32
}

const cgroupUDPClientShardCount = 16

type cgroupUDPClientShard struct {
	access  sync.RWMutex
	clients map[netip.AddrPort]*cgroupUDPClientState
}

type cgroupUDPClientState struct {
	access               sync.RWMutex
	connectedBinding     atomic.Pointer[cgroupUDPRedirectBinding]
	connected            bool
	connectedDestination netip.AddrPort
	sourceMAC            net.HardwareAddr
	bindings             map[netip.AddrPort]cgroupUDPRedirectBinding
	originals            map[netip.Addr]udpOriginalDestination
	replyAliasCount      uint16
}

type cgroupUDPRedirectBinding struct {
	address    netip.Addr
	packetInfo []byte
	connected  bool
	reference  udpRedirectReference
	sharedFlow *commonEBPF.SharedNetworkFlowHandle
	replyAlias bool
}

type udpRedirectReference struct {
	client  netip.AddrPort
	address netip.Addr
}

type udpRedirectRelease struct {
	reference  udpRedirectReference
	sharedFlow *commonEBPF.SharedNetworkFlowHandle
}

type udpOriginalDestination struct {
	original   commonEBPF.OriginalDestination
	sharedFlow *commonEBPF.SharedNetworkFlowHandle
	replyAlias bool
}

const cgroupUDPReplyAliasLimit = 64

func (t *cgroupUDPClientTable) load(client netip.AddrPort) (*cgroupUDPClientState, bool) {
	shard := t.clientShard(client)
	shard.access.RLock()
	clientState, loaded := shard.clients[client]
	shard.access.RUnlock()
	return clientState, loaded
}

func (t *cgroupUDPClientTable) current(client netip.AddrPort, expectedState *cgroupUDPClientState) bool {
	clientState, loaded := t.load(client)
	return loaded && clientState == expectedState
}

func (t *cgroupUDPClientTable) loadOrCreate(client netip.AddrPort) *cgroupUDPClientState {
	if clientState, loaded := t.load(client); loaded {
		return clientState
	}
	shard := t.clientShard(client)
	shard.access.Lock()
	defer shard.access.Unlock()
	return shard.loadOrCreateLocked(client)
}

func (s *cgroupUDPClientShard) loadOrCreateLocked(client netip.AddrPort) *cgroupUDPClientState {
	if clientState, loaded := s.clients[client]; loaded {
		return clientState
	}
	if s.clients == nil {
		s.clients = make(map[netip.AddrPort]*cgroupUDPClientState)
	}
	clientState := &cgroupUDPClientState{
		bindings:  make(map[netip.AddrPort]cgroupUDPRedirectBinding),
		originals: make(map[netip.Addr]udpOriginalDestination),
	}
	s.clients[client] = clientState
	return clientState
}

func (t *cgroupUDPClientTable) clientShard(client netip.AddrPort) *cgroupUDPClientShard {
	port := client.Port()
	index := (port ^ port>>8) & (cgroupUDPClientShardCount - 1)
	return &t.clientShards[index]
}

func (t *cgroupUDPClientTable) cachedOriginal(client netip.AddrPort, redirectAddress netip.Addr) (udpOriginalDestination, bool) {
	original, _, loaded := t.cachedPacketState(client, redirectAddress)
	return original, loaded
}

func (t *cgroupUDPClientTable) cachedPacketState(
	client netip.AddrPort,
	redirectAddress netip.Addr,
) (udpOriginalDestination, bool, bool) {
	clientState, loaded := t.load(client)
	if !loaded {
		return udpOriginalDestination{}, false, false
	}
	clientState.access.RLock()
	original, loaded := clientState.originals[redirectAddress]
	if !loaded {
		clientState.access.RUnlock()
		return udpOriginalDestination{}, false, false
	}
	binding, bindingLoaded := clientState.bindings[original.original.Destination]
	bindingReady := bindingLoaded &&
		binding.address == redirectAddress &&
		binding.connected == original.original.ConnectedUDP
	clientState.access.RUnlock()
	return original, bindingReady, true
}

func (t *cgroupUDPClientTable) setBinding(
	client netip.AddrPort,
	destination netip.AddrPort,
	redirectAddress netip.Addr,
	connected bool,
) []netip.Addr {
	releases, _ := t.setBindingState(
		client,
		redirectAddress,
		udpRedirectReference{address: redirectAddress},
		udpOriginalDestination{
			original: commonEBPF.OriginalDestination{
				Destination:  destination,
				ConnectedUDP: connected,
			},
		},
	)
	addresses := make([]netip.Addr, 0, len(releases))
	for _, release := range releases {
		addresses = append(addresses, release.reference.address)
	}
	return addresses
}

func (t *cgroupUDPClientTable) setSharedBinding(
	client netip.AddrPort,
	original commonEBPF.OriginalDestination,
	redirectAddress netip.Addr,
	flow *commonEBPF.SharedNetworkFlowHandle,
) ([]udpRedirectRelease, bool) {
	return t.setBindingState(
		client,
		redirectAddress,
		udpRedirectReference{client: client, address: redirectAddress},
		udpOriginalDestination{
			original:   original,
			sharedFlow: flow,
		},
	)
}

func (t *cgroupUDPClientTable) setReplyBinding(
	client netip.AddrPort,
	expectedState *cgroupUDPClientState,
	destination netip.AddrPort,
	redirectAddress netip.Addr,
) ([]netip.Addr, bool) {
	releases, installed := t.setExistingBindingState(
		client,
		expectedState,
		redirectAddress,
		udpRedirectReference{address: redirectAddress},
		udpOriginalDestination{
			original:   commonEBPF.OriginalDestination{Destination: destination},
			replyAlias: true,
		},
	)
	addresses := make([]netip.Addr, 0, len(releases))
	for _, release := range releases {
		addresses = append(addresses, release.reference.address)
	}
	return addresses, installed
}

func (t *cgroupUDPClientTable) setSharedReplyBinding(
	client netip.AddrPort,
	expectedState *cgroupUDPClientState,
	original commonEBPF.OriginalDestination,
	redirectAddress netip.Addr,
	flow *commonEBPF.SharedNetworkFlowHandle,
) ([]udpRedirectRelease, bool) {
	return t.setExistingBindingState(
		client,
		expectedState,
		redirectAddress,
		udpRedirectReference{client: client, address: redirectAddress},
		udpOriginalDestination{original: original, sharedFlow: flow, replyAlias: true},
	)
}

func (t *cgroupUDPClientTable) setExistingBindingState(
	client netip.AddrPort,
	expectedState *cgroupUDPClientState,
	redirectAddress netip.Addr,
	reference udpRedirectReference,
	original udpOriginalDestination,
) ([]udpRedirectRelease, bool) {
	shard := t.clientShard(client)
	shard.access.RLock()
	defer shard.access.RUnlock()
	if shard.clients[client] != expectedState {
		return nil, false
	}
	return t.setClientBinding(expectedState, redirectAddress, reference, original)
}

func (t *cgroupUDPClientTable) setBindingState(
	client netip.AddrPort,
	redirectAddress netip.Addr,
	reference udpRedirectReference,
	original udpOriginalDestination,
) ([]udpRedirectRelease, bool) {
	shard := t.clientShard(client)
	shard.access.RLock()
	clientState, loaded := shard.clients[client]
	if loaded {
		released, installed := t.setClientBinding(clientState, redirectAddress, reference, original)
		shard.access.RUnlock()
		return released, installed
	}
	shard.access.RUnlock()

	shard.access.Lock()
	clientState = shard.loadOrCreateLocked(client)
	released, installed := t.setClientBinding(clientState, redirectAddress, reference, original)
	shard.access.Unlock()
	return released, installed
}

func (t *cgroupUDPClientTable) setClientBinding(
	clientState *cgroupUDPClientState,
	redirectAddress netip.Addr,
	reference udpRedirectReference,
	original udpOriginalDestination,
) ([]udpRedirectRelease, bool) {
	destination := original.original.Destination
	connected := original.original.ConnectedUDP
	clientState.access.RLock()
	current, loaded := clientState.bindings[destination]
	clientState.access.RUnlock()
	if loaded && current.address == redirectAddress && current.connected == connected &&
		current.replyAlias == original.replyAlias {
		return nil, false
	}

	clientState.access.Lock()
	defer clientState.access.Unlock()
	current, loaded = clientState.bindings[destination]
	if loaded && current.address == redirectAddress && current.connected == connected &&
		current.replyAlias == original.replyAlias {
		return nil, false
	}
	if original.replyAlias && (!loaded || !current.replyAlias) && clientState.replyAliasCount >= cgroupUDPReplyAliasLimit {
		return nil, false
	}
	clientState.originals[redirectAddress] = original
	if len(original.original.SourceMAC) != 0 {
		clientState.sourceMAC = append(clientState.sourceMAC[:0], original.original.SourceMAC...)
	}
	binding := cgroupUDPRedirectBinding{
		address:    redirectAddress,
		packetInfo: sourcePacketInfo(redirectAddress),
		connected:  connected,
		reference:  reference,
		sharedFlow: original.sharedFlow,
		replyAlias: original.replyAlias,
	}
	clientState.bindings[destination] = binding
	if original.replyAlias && (!loaded || !current.replyAlias) {
		clientState.replyAliasCount++
	} else if !original.replyAlias && loaded && current.replyAlias {
		clientState.replyAliasCount--
	}
	if clientState.connected && clientState.connectedDestination == destination {
		connectedBinding := binding
		clientState.connectedBinding.Store(&connectedBinding)
	}
	if loaded && current.address != redirectAddress {
		clientState.deleteUnusedOriginalLocked(current.address)
	}

	t.redirectAccess.Lock()
	defer t.redirectAccess.Unlock()
	if !connected {
		t.retainRedirectLocked(reference)
	}
	if loaded && !current.connected && t.releaseRedirectLocked(current.reference) {
		return []udpRedirectRelease{{
			reference:  current.reference,
			sharedFlow: current.sharedFlow,
		}}, true
	}
	return nil, true
}

func (s *cgroupUDPClientState) deleteUnusedOriginalLocked(address netip.Addr) {
	for _, binding := range s.bindings {
		if binding.address == address {
			return
		}
	}
	delete(s.originals, address)
}

func (t *cgroupUDPClientTable) delete(client netip.AddrPort, expectedState *cgroupUDPClientState) []netip.Addr {
	releases := t.deleteClient(client, expectedState)
	addresses := make([]netip.Addr, 0, len(releases))
	for _, release := range releases {
		addresses = append(addresses, release.reference.address)
	}
	return addresses
}

func (t *cgroupUDPClientTable) deleteShared(client netip.AddrPort, expectedState *cgroupUDPClientState) []udpRedirectRelease {
	return t.deleteClient(client, expectedState)
}

func (t *cgroupUDPClientTable) deleteClient(client netip.AddrPort, expectedState *cgroupUDPClientState) []udpRedirectRelease {
	shard := t.clientShard(client)
	shard.access.Lock()
	defer shard.access.Unlock()
	if shard.clients[client] != expectedState {
		return nil
	}
	delete(shard.clients, client)

	expectedState.access.Lock()
	defer expectedState.access.Unlock()
	t.redirectAccess.Lock()
	defer t.redirectAccess.Unlock()
	var released []udpRedirectRelease
	for _, binding := range expectedState.bindings {
		if !binding.connected && t.releaseRedirectLocked(binding.reference) {
			released = append(released, udpRedirectRelease{
				reference:  binding.reference,
				sharedFlow: binding.sharedFlow,
			})
		}
	}
	clear(expectedState.bindings)
	clear(expectedState.originals)
	expectedState.replyAliasCount = 0
	expectedState.connectedBinding.Store(nil)
	return released
}

func (t *cgroupUDPClientTable) retainRedirectLocked(reference udpRedirectReference) {
	if t.redirectReferences == nil {
		t.redirectReferences = make(map[udpRedirectReference]uint32)
	}
	t.redirectReferences[reference]++
}

func (t *cgroupUDPClientTable) releaseRedirectLocked(reference udpRedirectReference) bool {
	references := t.redirectReferences[reference]
	if references > 1 {
		t.redirectReferences[reference] = references - 1
		return false
	}
	if references == 1 {
		delete(t.redirectReferences, reference)
		return true
	}
	return false
}

func (s *cgroupUDPClientState) redirectBinding(destination netip.AddrPort) (cgroupUDPRedirectBinding, bool) {
	if binding := s.connectedBinding.Load(); binding != nil {
		return *binding, true
	}
	s.access.RLock()
	if s.connected {
		destination = s.connectedDestination
	}
	binding, loaded := s.bindings[destination]
	s.access.RUnlock()
	return binding, loaded
}

func (s *cgroupUDPClientState) replyTemplate(destination netip.AddrPort, shared bool) (cgroupUDPRedirectBinding, bool) {
	s.access.RLock()
	defer s.access.RUnlock()
	if s.replyAliasCount >= cgroupUDPReplyAliasLimit {
		return cgroupUDPRedirectBinding{}, false
	}
	for _, binding := range s.bindings {
		if binding.address.Is4() == destination.Addr().Is4() && (!shared || binding.sharedFlow != nil) {
			return binding, true
		}
	}
	return cgroupUDPRedirectBinding{}, false
}

func (s *cgroupUDPClientState) sourceMACAddress() net.HardwareAddr {
	s.access.RLock()
	defer s.access.RUnlock()
	return append(net.HardwareAddr(nil), s.sourceMAC...)
}

func sourcePacketInfo(address netip.Addr) []byte {
	if address.Is4() {
		return (&ipv4.ControlMessage{Src: net.IP(address.AsSlice())}).Marshal()
	}
	return (&ipv6.ControlMessage{Src: net.IP(address.AsSlice())}).Marshal()
}

func (s *cgroupUDPClientState) setConnected(connected bool, destination netip.AddrPort) {
	s.access.Lock()
	s.connected = connected
	if connected {
		s.connectedDestination = destination
		if binding, loaded := s.bindings[destination]; loaded {
			connectedBinding := binding
			s.connectedBinding.Store(&connectedBinding)
		} else {
			s.connectedBinding.Store(nil)
		}
	} else {
		s.connectedDestination = netip.AddrPort{}
		s.connectedBinding.Store(nil)
	}
	s.access.Unlock()
}

func (s *cgroupUDPClientState) isConnected() bool {
	s.access.RLock()
	connected := s.connected
	s.access.RUnlock()
	return connected
}
