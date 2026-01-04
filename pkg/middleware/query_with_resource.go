package middleware

import cedar "github.com/cedar-policy/cedar-go"

type QueryWithResourceNext func(*Context) error

type QueryWithResourceMiddleware func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next QueryWithResourceNext) error

type QueryWithResourceChain struct {
	middlewares []QueryWithResourceMiddleware
}

func NewQueryWithResourceChain(middlewares ...QueryWithResourceMiddleware) *QueryWithResourceChain {
	return &QueryWithResourceChain{middlewares: middlewares}
}

func (c *QueryWithResourceChain) Execute(ctx *Context, action cedar.EntityUID, resource cedar.Entity, final QueryWithResourceNext) error {
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
