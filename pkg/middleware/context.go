package middleware

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

type Context struct {
	context.Context
	events []any
}

type ContextOpt func(*Context)

type principalKey struct{}
type dispatcherKey struct{}
type metricsCollectorKey struct{}

func WithPrincipal(p cedar.EntityUID) ContextOpt {
	return func(c *Context) {
		c.Context = context.WithValue(c.Context, principalKey{}, p)
	}
}

func WithTransaction(tx *bstore.Tx) ContextOpt {
	return func(c *Context) {
		c.Context = store.WithTx(c.Context, tx)
	}
}

func WithStore(s *store.Store) ContextOpt {
	return func(c *Context) {
		c.Context = store.WithStore(c.Context, s)
	}
}

func WithEventDispatcher(d EventDispatcher) ContextOpt {
	return func(c *Context) {
		c.Context = context.WithValue(c.Context, dispatcherKey{}, d)
	}
}

func DispatcherFromContext(ctx context.Context) (EventDispatcher, bool) {
	if ctx == nil {
		return nil, false
	}
	d, ok := ctx.Value(dispatcherKey{}).(EventDispatcher)
	return d, ok
}

func WithMetrics(m telemetry.Metrics) ContextOpt {
	return func(c *Context) {
		c.Context = telemetry.WithMetrics(c.Context, m)
	}
}

func MetricsCollectorFromContext(ctx context.Context) (*MetricsCollector, bool) {
	if ctx == nil {
		return nil, false
	}
	mc, ok := ctx.Value(metricsCollectorKey{}).(*MetricsCollector)
	return mc, ok
}

func WithMetricsCollector(mc *MetricsCollector) ContextOpt {
	return func(c *Context) {
		c.Context = context.WithValue(c.Context, metricsCollectorKey{}, mc)
	}
}

func ContextWithPrincipal(ctx context.Context, p cedar.EntityUID) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, principalKey{}, p)
}

func WithAnonymousPrincipal() ContextOpt {
	return WithPrincipal(cedar.NewEntityUID(cedar.EntityType("Mixology::Actor"), cedar.String("anonymous")))
}

func NewContext(parent context.Context, opts ...ContextOpt) *Context {
	if parent == nil {
		parent = context.Background()
	}

	c := &Context{
		Context: parent,
		events:  make([]any, 0, 4),
	}
	for _, opt := range opts {
		opt(c)
	}

	if _, ok := c.Context.Value(principalKey{}).(cedar.EntityUID); !ok {
		WithAnonymousPrincipal()(c)
	}

	return c
}

func (c *Context) AddEvent(event any) {
	c.events = append(c.events, event)
}

func (c *Context) Events() []any {
	return c.events
}

func (c *Context) Principal() cedar.EntityUID {
	if p, ok := c.Context.Value(principalKey{}).(cedar.EntityUID); ok {
		return p
	}
	return cedar.NewEntityUID(cedar.EntityType("Mixology::Actor"), cedar.String("anonymous"))
}

func (c *Context) Transaction() (*bstore.Tx, bool) {
	return store.TxFromContext(c.Context)
}
