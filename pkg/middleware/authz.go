package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/authz"
	cedar "github.com/cedar-policy/cedar-go"
)

// QueryAuthorize authorizes a query action using the AuthZ subchain for observability.
// The subchain handles logging and metrics around the authorization check.
func QueryAuthorize() QueryMiddleware {
	return func(ctx *Context, action cedar.EntityUID, next QueryNext) error {
		err := DefaultAuthZChain.Execute(ctx, action, func() error {
			return authz.Authorize(ctx.Principal(), action)
		})
		if err != nil {
			return err
		}
		return next(ctx)
	}
}

// QueryWithResourceAuthorize authorizes a query action with a resource using the AuthZ subchain.
// The subchain handles logging and metrics around the authorization check.
func QueryWithResourceAuthorize() QueryWithResourceMiddleware {
	return func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next QueryWithResourceNext) error {
		err := DefaultAuthZChain.Execute(ctx, action, func() error {
			return authz.AuthorizeWithEntity(ctx.Principal(), action, resource)
		})
		if err != nil {
			return err
		}
		return next(ctx)
	}
}

// CommandAuthorize authorizes a command action with a resource using the AuthZ subchain.
// The subchain handles logging and metrics around the authorization check.
func CommandAuthorize() CommandMiddleware {
	return func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next CommandNext) error {
		err := DefaultAuthZChain.Execute(ctx, action, func() error {
			return authz.AuthorizeWithEntity(ctx.Principal(), action, resource)
		})
		if err != nil {
			return err
		}
		return next(ctx)
	}
}
