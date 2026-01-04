package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

type DAO struct{}

func New() *DAO { return &DAO{} }

func (d *DAO) Get(ctx context.Context, id string) (models.Ingredient, bool, error) {
	var (
		out   models.Ingredient
		found bool
	)

	err := d.read(ctx, func(tx *bstore.Tx) error {
		out = models.Ingredient{ID: id}
		if err := tx.Get(&out); err != nil {
			if err == bstore.ErrAbsent {
				out = models.Ingredient{}
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

func (d *DAO) List(ctx context.Context) ([]models.Ingredient, error) {
	var out []models.Ingredient
	err := d.read(ctx, func(tx *bstore.Tx) error {
		ingredients, err := bstore.QueryTx[models.Ingredient](tx).List()
		if err != nil {
			return err
		}
		out = ingredients
		return nil
	})
	return out, err
}

func (d *DAO) Insert(ctx context.Context, ingredient models.Ingredient) error {
	return d.write(ctx, func(tx *bstore.Tx) error {
		return tx.Insert(&ingredient)
	})
}

func (d *DAO) Update(ctx context.Context, ingredient models.Ingredient) error {
	return d.write(ctx, func(tx *bstore.Tx) error {
		return tx.Update(&ingredient)
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
