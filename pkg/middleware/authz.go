package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/authz"
	cedar "github.com/cedar-policy/cedar-go"
)

// AuthorizeQuery authorizes query operations, using the shared query resource
// when the operation does not carry a resource of its own.
func AuthorizeQuery() Middleware {
	return func(ctx *Context, op Operation, next Next) error {
		if op.HasResource() {
			if err := authz.AuthorizeWithEntity(ctx.Principal(), op.Action, op.Resource); err != nil {
				return err
			}
		} else {
			if err := authz.Authorize(ctx.Principal(), op.Action); err != nil {
				return err
			}
		}
		return next(ctx)
	}
}

// CommandHandler handles a loaded command resource and returns its resulting
// state.
type CommandHandler[In CedarEntity, Out CedarEntity] func(*Context, In) (Out, error)

// AuthorizeCommand authorizes both sides of a command state transition. The
// loaded resource is checked before the handler runs, and the resulting
// resource is checked before the result is returned.
func AuthorizeCommand[In CedarEntity, Out CedarEntity](action cedar.EntityUID, next CommandHandler[In, Out]) CommandHandler[In, Out] {
	return func(ctx *Context, in In) (Out, error) {
		var zero Out
		if err := authz.AuthorizeWithEntity(ctx.Principal(), action, in.CedarEntity()); err != nil {
			return zero, err
		}

		out, err := next(ctx, in)
		if err != nil {
			return zero, err
		}
		if err := authz.AuthorizeWithEntity(ctx.Principal(), action, out.CedarEntity()); err != nil {
			return zero, err
		}
		return out, nil
	}
}
