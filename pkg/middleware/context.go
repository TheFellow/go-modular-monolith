package middleware

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

type Context struct {
	context.Context
	events           []any
	principal        cedar.EntityUID
	store            *store.Store
	tx               *bstore.Tx
	dispatcher       EventDispatcher
	metricsCollector *MetricsCollector
	activity         *middlewareevents.Activity
	activityRecorder ActivityRecorder
}

type ContextConfig struct {
	Principal        cedar.EntityUID
	Store            *store.Store
	Dispatcher       EventDispatcher
	MetricsCollector *MetricsCollector
	ActivityRecorder ActivityRecorder
}

func NewContext(parent context.Context, config ContextConfig) *Context {
	if parent == nil {
		parent = context.Background()
	}

	c := &Context{
		Context:          parent,
		events:           make([]any, 0, 4),
		principal:        config.Principal,
		store:            config.Store,
		dispatcher:       config.Dispatcher,
		metricsCollector: config.MetricsCollector,
		activityRecorder: config.ActivityRecorder,
	}

	if c.principal.IsZero() {
		c.principal = authn.Anonymous()
	}

	return c
}

func (c *Context) WithTransaction(tx *bstore.Tx) *Context {
	derived := *c
	derived.Context = c.Context
	derived.events = make([]any, 0, 4)
	derived.tx = tx
	return &derived
}

func (c *Context) AddEvent(event any) {
	c.events = append(c.events, event)
}

func (c *Context) Events() []any {
	return c.events
}

func (c *Context) Principal() cedar.EntityUID {
	if c != nil && !c.principal.IsZero() {
		return c.principal
	}
	return authn.Anonymous()
}

func (c *Context) Store() (*store.Store, bool) {
	if c == nil || c.store == nil {
		return nil, false
	}
	return c.store, true
}

func (c *Context) Transaction() (*bstore.Tx, bool) {
	if c == nil || c.tx == nil {
		return nil, false
	}
	return c.tx, true
}

func (c *Context) Dispatcher() (EventDispatcher, bool) {
	if c == nil || c.dispatcher == nil {
		return nil, false
	}
	return c.dispatcher, true
}

func (c *Context) MetricsCollector() (*MetricsCollector, bool) {
	if c == nil || c.metricsCollector == nil {
		return nil, false
	}
	return c.metricsCollector, true
}

func (c *Context) Activity() (*middlewareevents.Activity, bool) {
	if c == nil || c.activity == nil {
		return nil, false
	}
	return c.activity, true
}

func (c *Context) ActivityRecorder() (ActivityRecorder, bool) {
	if c == nil || c.activityRecorder == nil {
		return nil, false
	}
	return c.activityRecorder, true
}

// HandlerContext is a restricted context passed to event handlers.
// Handlers are leaf nodes — they can read data, persist changes within
// their own domain, and touch entities, but they cannot emit new events.
// This no-cascading rule is enforced at compile time.
type HandlerContext struct {
	context.Context
	ctx *Context
}

func NewHandlerContext(ctx *Context) *HandlerContext {
	return &HandlerContext{Context: ctx.Context, ctx: ctx}
}

func (h *HandlerContext) Transaction() (*bstore.Tx, bool) {
	return h.ctx.Transaction()
}

func (h *HandlerContext) Store() (*store.Store, bool) {
	return h.ctx.Store()
}

func (h *HandlerContext) TouchEntity(uid cedar.EntityUID) {
	h.ctx.TouchEntity(uid)
}

func (h *HandlerContext) Principal() cedar.EntityUID {
	return h.ctx.Principal()
}
