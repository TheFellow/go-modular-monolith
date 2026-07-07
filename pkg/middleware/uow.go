package middleware

import (
	"github.com/mjl-/bstore"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func UnitOfWork() Middleware {
	return func(ctx *Context, op Operation, next Next) error {
		if op.Kind != OperationKindCommand {
			return next(ctx)
		}

		s, ok := store.FromContext(ctx.Context)
		if !ok || s == nil {
			return errors.Internalf("store missing from context")
		}

		return s.Write(ctx, func(tx *bstore.Tx) error {
			txCtx := NewContext(ctx, WithTransaction(tx))
			return next(txCtx)
		})
	}
}
