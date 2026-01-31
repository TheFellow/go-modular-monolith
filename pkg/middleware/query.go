package middleware

import "github.com/cedar-policy/cedar-go"

type QueryNext func(*Context) error

type QueryMiddleware func(ctx *Context, action cedar.EntityUID, next QueryNext) error

type QueryChain struct {
	middlewares []QueryMiddleware
}

func NewQueryChain(middlewares ...QueryMiddleware) *QueryChain {
	return &QueryChain{middlewares: middlewares}
}

func (c *QueryChain) Execute(ctx *Context, action cedar.EntityUID, final QueryNext) error {
	next := final
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		m := c.middlewares[i]
		prev := next
		next = func(inner *Context) error {
			return m(inner, action, prev)
		}
	}
	return next(ctx)
}
