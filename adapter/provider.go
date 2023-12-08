package adapter

import (
	"context"
	"time"
)

type OutboundProvider interface {
	Tag() string
	Path() string
	Type() string
	Outbounds() []Outbound
	Outbound(tag string) (Outbound, bool)
	UpdateTime() time.Time

	Start()
	Close() error
	PRLock()
	PRUnlock()
	Healthcheck(ctx context.Context, link string, force bool) (map[string]uint16, error)
	SubscriptionInfo() map[string]uint64
	UpdateProvider(ctx context.Context, router Router, force bool) error
	UpdateOutboundByTag()
}
