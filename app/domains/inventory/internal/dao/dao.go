package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

type DAO struct{}

func New() *DAO { return &DAO{} }

func (d *DAO) Get(ctx context.Context, ingredientID string) (models.Stock, bool, error) {
	var (
		out   models.Stock
		found bool
	)

	err := d.read(ctx, func(tx *bstore.Tx) error {
		out = models.Stock{IngredientID: ingredientID}
		if err := tx.Get(&out); err != nil {
			if err == bstore.ErrAbsent {
				out = models.Stock{}
				found = false
				return nil
			}
			return err
		}
		found = true
		return nil
	})
	return out, found, err
}

func (d *DAO) List(ctx context.Context) ([]models.Stock, error) {
	var out []models.Stock
	err := d.read(ctx, func(tx *bstore.Tx) error {
		stock, err := bstore.QueryTx[models.Stock](tx).List()
		if err != nil {
			return err
		}
		out = stock
		return nil
	})
	return out, err
}

func (d *DAO) Upsert(ctx context.Context, stock models.Stock) error {
	return d.write(ctx, func(tx *bstore.Tx) error {
		if err := tx.Update(&stock); err != nil {
			if err == bstore.ErrAbsent {
				return tx.Insert(&stock)
			}
			return err
		}
		return nil
	})
}

func (d *DAO) read(ctx context.Context, f func(*bstore.Tx) error) error {
	if tx, ok := store.TxFromContext(ctx); ok && tx != nil {
		return f(tx)
	}
	if store.DB == nil {
		return errors.Internalf("store not initialized")
	}
	return store.DB.Read(ctx, func(tx *bstore.Tx) error {
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
