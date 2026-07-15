package middleware

import (
	"github.com/mjl-/bstore"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func UnitOfWork() Middleware {
	return func(ctx *Context, op Operation, next Next) error {
		if op.Kind != OperationKindCommand {
			return next(ctx)
		}

		s, ok := ctx.Store()
		if !ok || s == nil {
			return errors.Internalf("store missing from context")
		}

		return s.Write(ctx, func(tx *bstore.Tx) error {
			txCtx := ctx.WithTransaction(tx)
			return next(txCtx)
		})
	}
}
