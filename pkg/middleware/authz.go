package middleware

import cedar "github.com/cedar-policy/cedar-go"

func QueryAuthZ() QueryMiddleware {
	return func(ctx *Context, _ cedar.EntityUID, next QueryNext) error {
		return next(ctx)
	}
}

func CommandAuthZ() CommandMiddleware {
	return func(ctx *Context, _ cedar.EntityUID, _ cedar.Entity, next CommandNext) error {
		return next(ctx)
	}
}
