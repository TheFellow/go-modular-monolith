package middleware

import cedar "github.com/cedar-policy/cedar-go"

type CommandNext func(*Context) error

type CommandMiddleware func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next CommandNext) error

type CommandChain struct {
	middlewares []CommandMiddleware
}

func NewCommandChain(middlewares ...CommandMiddleware) *CommandChain {
	return &CommandChain{middlewares: middlewares}
}

func (c *CommandChain) Execute(ctx *Context, action cedar.EntityUID, resource cedar.Entity, final CommandNext) error {
	next := final
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		m := c.middlewares[i]
		prev := next
		next = func(inner *Context) error {
			return m(inner, action, resource, prev)
		}
	}
	return next(ctx)
}
