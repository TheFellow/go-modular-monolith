package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

type DAO struct{}

func New() *DAO { return &DAO{} }

func (d *DAO) Get(ctx context.Context, id string) (models.Order, bool, error) {
	var (
		out   models.Order
		found bool
	)

	err := d.read(ctx, func(tx *bstore.Tx) error {
		out = models.Order{ID: id}
		if err := tx.Get(&out); err != nil {
			if err == bstore.ErrAbsent {
				out = models.Order{}
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

func (d *DAO) List(ctx context.Context) ([]models.Order, error) {
	var out []models.Order
	err := d.read(ctx, func(tx *bstore.Tx) error {
		orders, err := bstore.QueryTx[models.Order](tx).List()
		if err != nil {
			return err
		}
		out = orders
		return nil
	})
	return out, err
}

func (d *DAO) Insert(ctx context.Context, order models.Order) error {
	return d.write(ctx, func(tx *bstore.Tx) error {
		return tx.Insert(&order)
	})
}

func (d *DAO) Update(ctx context.Context, order models.Order) error {
	return d.write(ctx, func(tx *bstore.Tx) error {
		return tx.Update(&order)
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
