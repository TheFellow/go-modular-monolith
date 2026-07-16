package middleware

import (
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
