package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
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

// CommandAuthorize authorizes a command action with a resource.
// Observability is handled by the top-level command logging/metrics middleware.
func CommandAuthorize() CommandMiddleware {
	return func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next CommandNext) error {
		input, ok := ctx.InputEntity()
		if !ok {
			loader, ok := commandLoaderFromContext(ctx.Context)
			if !ok {
				return errors.Internalf("command loader missing")
			}
			loaded, err := loader(ctx)
			if err != nil {
				return err
			}
			ctx.setInputEntity(loaded)
			input = loaded
		}

		if err := authz.AuthorizeWithEntity(ctx.Principal(), action, input.CedarEntity()); err != nil {
			return err
		}

		if err := next(ctx); err != nil {
			return err
		}

		output, ok := ctx.OutputEntity()
		if !ok {
			return errors.Internalf("command output missing")
		}
		if err := authz.AuthorizeWithEntity(ctx.Principal(), action, output.CedarEntity()); err != nil {
			return err
		}
		return nil
	}
}
