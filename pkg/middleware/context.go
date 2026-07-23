package middleware

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

type Context struct {
	context.Context
	events    []any
	principal cedar.EntityUID
	tx        *bstore.Tx
	activity  *middlewareevents.Activity
}

func NewContext(parent context.Context) *Context {
	var tx *bstore.Tx
	if parentCtx, ok := parent.(*Context); ok {
		tx = parentCtx.tx
	}
	principal := authn.FromContext(parent)
	parent = log.ToContext(parent, log.FromContext(parent).With(log.Actor(principal)))
	c := &Context{
		Context:   parent,
		events:    make([]any, 0, 4),
		principal: principal,
		tx:        tx,
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

func (c *Context) Transaction() (*bstore.Tx, bool) {
	if c == nil || c.tx == nil {
		return nil, false
	}
	return c.tx, true
}

func (c *Context) Activity() (*middlewareevents.Activity, bool) {
	if c == nil || c.activity == nil {
		return nil, false
	}
	return c.activity, true
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

func (h *HandlerContext) TouchEntity(uid cedar.EntityUID) {
	h.ctx.TouchEntity(uid)
}

func (h *HandlerContext) Principal() cedar.EntityUID {
	return h.ctx.Principal()
}
