package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/authz"
	cedar "github.com/cedar-policy/cedar-go"
)

func QueryAuthZ() QueryMiddleware {
	return func(ctx *Context, action cedar.EntityUID, next QueryNext) error {
		if err := authz.Authorize(ctx, ctx.Principal(), action); err != nil {
			return err
		}
		return next(ctx)
	}
}

func QueryAuthZWithResource() QueryWithResourceMiddleware {
	return func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next QueryWithResourceNext) error {
		if err := authz.AuthorizeWithEntity(ctx, ctx.Principal(), action, resource); err != nil {
			return err
		}
		return next(ctx)
	}
}

func CommandAuthZ() CommandMiddleware {
	return func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next CommandNext) error {
		if err := authz.AuthorizeWithEntity(ctx, ctx.Principal(), action, resource); err != nil {
			return err
		}
		return next(ctx)
	}
}
