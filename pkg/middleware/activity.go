package middleware

import (
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	cedar "github.com/cedar-policy/cedar-go"
)

// ActivityRecorder persists completed command activity outside domain event dispatch.
type ActivityRecorder interface {
	RecordActivity(ctx *Context, activity middlewareevents.Activity) error
}

func (c *Context) TouchEntity(uid cedar.EntityUID) {
	if c == nil {
		return
	}
	if a, ok := c.Activity(); ok {
		a.Touch(uid)
	}
}
