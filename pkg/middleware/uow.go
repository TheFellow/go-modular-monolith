package middleware

import (
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func UnitOfWork() CommandMiddleware {
	return func(ctx *Context, _ cedar.EntityUID, next CommandNext) error {
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
