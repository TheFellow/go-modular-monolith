package middleware

import (
	"context"

	cedar "github.com/cedar-policy/cedar-go"

	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
)

type activityKey struct{}

func WithActivity(a *middlewareevents.Activity) ContextOpt {
	return func(c *Context) {
		if a == nil {
			return
		}
		c.Context = context.WithValue(c.Context, activityKey{}, a)
	}
}

func ActivityFromContext(ctx context.Context) (*middlewareevents.Activity, bool) {
	if ctx == nil {
		return nil, false
	}
	a, ok := ctx.Value(activityKey{}).(*middlewareevents.Activity)
	return a, ok
}

func (c *Context) TouchEntity(uid cedar.EntityUID) {
	if c == nil {
		return
	}
	if a, ok := ActivityFromContext(c.Context); ok {
		a.Touch(uid)
	}
}

