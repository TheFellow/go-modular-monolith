package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/authz"
	cedar "github.com/cedar-policy/cedar-go"
)

// QueryAuthorize authorizes a query action using an AuthZ subchain for observability.
// Pass AuthZ middleware (e.g., AuthZLogging/AuthZMetrics) explicitly to avoid hidden defaults.
func QueryAuthorize(authzMiddlewares ...AuthZMiddleware) QueryMiddleware {
	authzChain := NewAuthZChain(authzMiddlewares...)
	return func(ctx *Context, action cedar.EntityUID, next QueryNext) error {
		err := authzChain.Execute(ctx, action, func() error {
			return authz.Authorize(ctx.Principal(), action)
		})
		if err != nil {
			return err
		}
		return next(ctx)
	}
}

// QueryWithResourceAuthorize authorizes a query action with a resource using an AuthZ subchain.
// Pass AuthZ middleware (e.g., AuthZLogging/AuthZMetrics) explicitly to avoid hidden defaults.
func QueryWithResourceAuthorize(authzMiddlewares ...AuthZMiddleware) QueryWithResourceMiddleware {
	authzChain := NewAuthZChain(authzMiddlewares...)
	return func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next QueryWithResourceNext) error {
		err := authzChain.Execute(ctx, action, func() error {
			return authz.AuthorizeWithEntity(ctx.Principal(), action, resource)
		})
		if err != nil {
			return err
		}
		return next(ctx)
	}
}

// CommandAuthorize authorizes a command action with a resource using an AuthZ subchain.
// Pass AuthZ middleware (e.g., AuthZLogging/AuthZMetrics) explicitly to avoid hidden defaults.
func CommandAuthorize(authzMiddlewares ...AuthZMiddleware) CommandMiddleware {
	authzChain := NewAuthZChain(authzMiddlewares...)
	return func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next CommandNext) error {
		err := authzChain.Execute(ctx, action, func() error {
			return authz.AuthorizeWithEntity(ctx.Principal(), action, resource)
		})
		if err != nil {
			return err
		}
		return next(ctx)
	}
}
