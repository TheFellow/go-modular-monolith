package middleware

import (
	"fmt"
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	cedar "github.com/cedar-policy/cedar-go"
)

func (c *Context) TouchEntity(uid cedar.EntityUID) {
	if c == nil {
		return
	}
	if a, ok := c.Activity(); ok {
		a.Touch(uid)
	}
}

// Change formats the domain-selected values carried by an audit effect.
func Change(field string, before, after any) middlewareevents.Change {
	return middlewareevents.Change{Field: field, Before: fmt.Sprint(before), After: fmt.Sprint(after)}
}
