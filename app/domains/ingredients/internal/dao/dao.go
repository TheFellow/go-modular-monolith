package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

type DAO struct{}

func New() *DAO { return &DAO{} }

func init() {
	store.RegisterTypes(IngredientRow{})
}

func (d *DAO) read(ctx context.Context, f func(*bstore.Tx) error) error {
	if tx, ok := store.TxFromContext(ctx); ok && tx != nil {
		return f(tx)
	}
	s, ok := store.FromContext(ctx)
	if !ok || s == nil {
		return errors.Internalf("store missing from context")
	}
	return s.Read(ctx, func(tx *bstore.Tx) error {
		return f(tx)
	})
}

func (d *DAO) write(ctx context.Context, f func(*bstore.Tx) error) error {
	tx, ok := store.TxFromContext(ctx)
	if !ok || tx == nil {
		return errors.Internalf("missing transaction")
	}
	return f(tx)
}
