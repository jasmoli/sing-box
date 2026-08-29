package outbound

import (
	"context"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing/common"
	E "github.com/sagernet/sing/common/exceptions"
)

var _ adapter.OutboundManager = (*Manager)(nil)

type Manager struct {
	registry        adapter.OutboundRegistry
	endpoint        adapter.EndpointManager
	defaultTag      string
	access          sync.RWMutex
	scope           *adapter.Scope
	stage           adapter.StartStage
	outbounds       []adapter.Outbound
	outboundByTag   map[string]adapter.Outbound
	defaultOutbound adapter.Outbound
	sortedOutbounds []adapter.Outbound
}

func NewManager(registry adapter.OutboundRegistry, endpoint adapter.EndpointManager, defaultTag string) *Manager {
	return &Manager{
		registry:      registry,
		endpoint:      endpoint,
		defaultTag:    defaultTag,
		outboundByTag: make(map[string]adapter.Outbound),
	}
}

func (m *Manager) Initialize(defaultOutboundFallback func() (adapter.Outbound, error)) error {
	m.access.Lock()
	defer m.access.Unlock()
	if m.defaultTag != "" && m.defaultOutbound == nil {
		defaultEndpoint, loaded := m.endpoint.Get(m.defaultTag)
		if !loaded {
			return E.New("default outbound not found: ", m.defaultTag)
		}
		m.defaultOutbound = defaultEndpoint
	}
	if m.defaultOutbound == nil {
		directOutbound, err := defaultOutboundFallback()
		if err != nil {
			return E.Cause(err, "create direct outbound for fallback")
		}
		m.outbounds = append(m.outbounds, directOutbound)
		m.outboundByTag[directOutbound.Tag()] = directOutbound
		m.defaultOutbound = directOutbound
	}
	sortedOutbounds, err := sortOutbounds(slices.Concat(m.outbounds, common.Map(m.endpoint.Endpoints(), func(it adapter.Endpoint) adapter.Outbound { return it })))
	if err != nil {
		return err
	}
	m.sortedOutbounds = sortedOutbounds
	return nil
}

func (m *Manager) Start(stage adapter.StartStage, scope *adapter.Scope) error {
	m.access.Lock()
	m.scope = scope
	m.stage = stage
	sortedOutbounds := m.sortedOutbounds
	m.access.Unlock()
	for _, outbound := range sortedOutbounds {
		if endpoint, isEndpoint := outbound.(adapter.Endpoint); isEndpoint {
			if stage == adapter.StartStateStart {
				err := m.endpoint.StartEndpoint(endpoint)
				if err != nil {
					return err
				}
			}
			continue
		}
		lifecycle, isLifecycle := outbound.(adapter.Lifecycle)
		if !isLifecycle {
			continue
		}
		name := "outbound/" + outbound.Type() + "[" + outbound.Tag() + "]"
		err := scope.Start(name, lifecycle, stage)
		if err != nil {
			return err
		}
	}
	return nil
}

func sortOutbounds(outbounds []adapter.Outbound) ([]adapter.Outbound, error) {
	sortedOutbounds := make([]adapter.Outbound, 0, len(outbounds))
	sorted := make(map[string]bool)
	for {
		canContinue := false
	sortOne:
		for _, outboundToSort := range outbounds {
			outboundTag := outboundToSort.Tag()
			if sorted[outboundTag] {
				continue
			}
			dependencies := outboundToSort.Dependencies()
			for _, dependency := range dependencies {
				if !sorted[dependency] {
					continue sortOne
				}
			}
			sorted[outboundTag] = true
			canContinue = true
			sortedOutbounds = append(sortedOutbounds, outboundToSort)
		}
		if len(sorted) == len(outbounds) {
			break
		}
		if canContinue {
			continue
		}
		currentOutbound := common.Find(outbounds, func(it adapter.Outbound) bool {
			return !sorted[it.Tag()]
		})
		var lintOutbound func(oTree []string, oCurrent adapter.Outbound) error
		lintOutbound = func(oTree []string, oCurrent adapter.Outbound) error {
			problemOutboundTag := common.Find(oCurrent.Dependencies(), func(it string) bool {
				return !sorted[it]
			})
			if common.Contains(oTree, problemOutboundTag) {
				return E.New("circular outbound dependency: ", strings.Join(oTree, " -> "), " -> ", problemOutboundTag)
			}
			problemOutbound := common.Find(outbounds, func(it adapter.Outbound) bool {
				return it.Tag() == problemOutboundTag
			})
			if problemOutbound == nil {
				return E.New("dependency[", problemOutboundTag, "] not found for outbound[", oCurrent.Tag(), "]")
			}
			return lintOutbound(append(oTree, problemOutboundTag), problemOutbound)
		}
		return nil, lintOutbound([]string{currentOutbound.Tag()}, currentOutbound)
	}
	return sortedOutbounds, nil
}

func (m *Manager) Outbounds() []adapter.Outbound {
	m.access.RLock()
	defer m.access.RUnlock()
	return m.outbounds
}

func (m *Manager) Outbound(tag string) (adapter.Outbound, bool) {
	m.access.RLock()
	outbound, found := m.outboundByTag[tag]
	m.access.RUnlock()
	if found {
		return outbound, true
	}
	return m.endpoint.Get(tag)
}

func (m *Manager) Default() adapter.Outbound {
	m.access.RLock()
	defer m.access.RUnlock()
	return m.defaultOutbound
}

func (m *Manager) Remove(tag string) error {
	m.access.Lock()
	defer m.access.Unlock()
	outbound, found := m.outboundByTag[tag]
	if !found {
		return os.ErrInvalid
	}
	delete(m.outboundByTag, tag)
	index := common.Index(m.outbounds, func(it adapter.Outbound) bool {
		return it == outbound
	})
	if index == -1 {
		panic("invalid outbound index")
	}
	m.outbounds = append(m.outbounds[:index], m.outbounds[index+1:]...)
	index = common.Index(m.sortedOutbounds, func(it adapter.Outbound) bool {
		return it == outbound
	})
	if index != -1 {
		m.sortedOutbounds = append(m.sortedOutbounds[:index], m.sortedOutbounds[index+1:]...)
	}
	if m.defaultOutbound == outbound {
		if len(m.outbounds) > 0 {
			m.defaultOutbound = m.outbounds[0]
		} else {
			m.defaultOutbound = nil
		}
	}
	return nil
}

func (m *Manager) Create(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, inboundType string, options any) error {
	if tag == "" {
		return os.ErrInvalid
	}
	outbound, err := m.registry.CreateOutbound(ctx, router, logger, tag, inboundType, options)
	if err != nil {
		return err
	}
	m.access.Lock()
	_, loaded := m.outboundByTag[tag]
	if loaded {
		m.access.Unlock()
		return E.New("duplicate outbound tag: ", tag)
	}
	m.outbounds = append(m.outbounds, outbound)
	m.outboundByTag[tag] = outbound
	if tag == m.defaultTag || (m.defaultTag == "" && m.defaultOutbound == nil) {
		m.defaultOutbound = outbound
	}
	m.sortedOutbounds = append(m.sortedOutbounds, outbound)
	scope := m.scope
	stage := m.stage
	m.access.Unlock()
	if scope != nil {
		err = m.startOutbound(scope, outbound, stage)
		if err != nil {
			_ = m.Remove(tag)
			return err
		}
	}
	return nil
}

func (m *Manager) startOutbound(scope *adapter.Scope, outbound adapter.Outbound, stage adapter.StartStage) error {
	lifecycle, isLifecycle := outbound.(adapter.Lifecycle)
	if !isLifecycle {
		return nil
	}
	name := "outbound/" + outbound.Type() + "[" + outbound.Tag() + "]"
	for _, startStage := range adapter.ListStartStages {
		if startStage > stage {
			break
		}
		err := scope.Start(name, lifecycle, startStage)
		if err != nil {
			return err
		}
	}
	return nil
}
