package middleware

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/uow"
	cedar "github.com/cedar-policy/cedar-go"
)

type Context struct {
	context.Context
	events []any
}

type ContextOpt func(*Context)

type principalKey struct{}
type uowKey struct{}

func WithPrincipal(p cedar.EntityUID) ContextOpt {
	return func(c *Context) {
		c.Context = context.WithValue(c.Context, principalKey{}, p)
	}
}

func WithAnonymousPrincipal() ContextOpt {
	return WithPrincipal(cedar.NewEntityUID(cedar.EntityType("Mixology::Actor"), cedar.String("anonymous")))
}

func NewContext(parent context.Context, opts ...ContextOpt) *Context {
	if parent == nil {
		parent = context.Background()
	}

	c := &Context{Context: parent}
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

func (c *Context) SetUnitOfWork(tx *uow.Tx) {
	c.Context = context.WithValue(c.Context, uowKey{}, tx)
}

func (c *Context) UnitOfWork() (*uow.Tx, bool) {
	tx, ok := c.Context.Value(uowKey{}).(*uow.Tx)
	return tx, ok
}
