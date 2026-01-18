package dao

import (
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

// Read executes f within a read transaction.
// If a transaction exists in context, uses it. Otherwise creates a new read tx.
func Read(ctx Context, f func(*bstore.Tx) error) error {
	if tx, ok := ctx.Transaction(); ok && tx != nil {
		return f(tx)
	}
	s, ok := store.FromContext(ctx)
	if !ok || s == nil {
		return errors.Internalf("store missing from context")
	}
	return s.Read(ctx, f)
}

// Write executes f within the existing write transaction.
// Requires a transaction in context (set by UnitOfWork middleware).
func Write(ctx Context, f func(*bstore.Tx) error) error {
	tx, ok := ctx.Transaction()
	if !ok || tx == nil {
		return errors.Internalf("missing transaction")
	}
	return f(tx)
}
