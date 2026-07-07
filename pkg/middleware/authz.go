package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/authz"
)

// Authorize authorizes query operations.
// Command authorization is handled inline by RunCommand.
func Authorize() Middleware {
	return func(ctx *Context, op Operation, next Next) error {
		if op.Kind != OperationKindQuery {
			return next(ctx)
		}

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
