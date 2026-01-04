package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

type DAO struct{}

func New() *DAO { return &DAO{} }

func (d *DAO) Get(ctx context.Context, id string) (models.Drink, bool, error) {
	var (
		out   models.Drink
		found bool
	)

	err := d.read(ctx, func(tx *bstore.Tx) error {
		out = models.Drink{ID: id}
		if err := tx.Get(&out); err != nil {
			if err == bstore.ErrAbsent {
				out = models.Drink{}
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

func (d *DAO) List(ctx context.Context) ([]models.Drink, error) {
	var out []models.Drink
	err := d.read(ctx, func(tx *bstore.Tx) error {
		drinks, err := bstore.QueryTx[models.Drink](tx).List()
		if err != nil {
			return err
		}
		out = drinks
		return nil
	})
	return out, err
}

func (d *DAO) Insert(ctx context.Context, drink models.Drink) error {
	return d.write(ctx, func(tx *bstore.Tx) error {
		return tx.Insert(&drink)
	})
}

func (d *DAO) Update(ctx context.Context, drink models.Drink) error {
	return d.write(ctx, func(tx *bstore.Tx) error {
		return tx.Update(&drink)
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
