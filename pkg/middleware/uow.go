package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func UnitOfWork(s *store.Store) Middleware {
	return func(ctx *Context, op Operation, next Next) error {
		if op.Kind != OperationKindCommand {
			return next(ctx)
		}

		if s == nil {
			return errors.Internalf("store missing from context")
		}
		if tx, ok := ctx.Transaction(); ok && tx != nil {
			return next(ctx)
		}

		return s.Write(ctx, func(tx *bstore.Tx) error {
			txCtx := ctx.WithTransaction(tx)
			return next(txCtx)
		})
	}
}
