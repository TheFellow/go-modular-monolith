package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/authz"
	cedar "github.com/cedar-policy/cedar-go"
)

// QueryAuthorize authorizes a query action.
// Observability is handled by the top-level query logging/metrics middleware.
func QueryAuthorize() QueryMiddleware {
	return func(ctx *Context, action cedar.EntityUID, next QueryNext) error {
		if err := authz.Authorize(ctx.Principal(), action); err != nil {
			return err
		}
		return next(ctx)
	}
}

// QueryWithResourceAuthorize authorizes a query action with a resource.
// Observability is handled by the top-level query logging/metrics middleware.
func QueryWithResourceAuthorize() QueryWithResourceMiddleware {
	return func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next QueryWithResourceNext) error {
		if err := authz.AuthorizeWithEntity(ctx.Principal(), action, resource); err != nil {
			return err
		}
		return next(ctx)
	}
}

// Command authorization is handled inline by RunCommand.
