package middleware

import (
	"context"

	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	cedar "github.com/cedar-policy/cedar-go"
)

type activityKey struct{}
type activityRecorderKey struct{}

// ActivityRecorder persists completed command activity outside domain event dispatch.
type ActivityRecorder interface {
	RecordActivity(ctx *Context, activity middlewareevents.Activity) error
}

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

// WithActivityRecorder attaches an activity recorder to the middleware context.
func WithActivityRecorder(r ActivityRecorder) ContextOpt {
	return func(c *Context) {
		c.Context = context.WithValue(c.Context, activityRecorderKey{}, r)
	}
}

// ActivityRecorderFromContext returns the activity recorder attached to ctx, if any.
func ActivityRecorderFromContext(ctx context.Context) (ActivityRecorder, bool) {
	if ctx == nil {
		return nil, false
	}
	r, ok := ctx.Value(activityRecorderKey{}).(ActivityRecorder)
	return r, ok
}

func (c *Context) TouchEntity(uid cedar.EntityUID) {
	if c == nil {
		return
	}
	if a, ok := ActivityFromContext(c.Context); ok {
		a.Touch(uid)
	}
}
