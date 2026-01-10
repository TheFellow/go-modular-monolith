package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

// ListFilter specifies optional filters for listing stock rows.
type ListFilter struct {
	IngredientID cedar.EntityUID
	MaxQuantity  optional.Value[float64]
	MinQuantity  optional.Value[float64]
}

func (d *DAO) List(ctx context.Context, filter ListFilter) ([]*models.Stock, error) {
	var out []*models.Stock
	err := d.read(ctx, func(tx *bstore.Tx) error {
		q := bstore.QueryTx[StockRow](tx)

		if string(filter.IngredientID.ID) != "" {
			q = q.FilterID(string(filter.IngredientID.ID))
		}
		if v, ok := filter.MaxQuantity.Unwrap(); ok {
			q = q.FilterLessEqual("Quantity", v)
		}
		if v, ok := filter.MinQuantity.Unwrap(); ok {
			q = q.FilterGreaterEqual("Quantity", v)
		}

		rows, err := q.List()
		if err != nil {
			return store.MapError(err, "list stock")
		}
		stock := make([]*models.Stock, 0, len(rows))
		for _, r := range rows {
			s := toModel(r)
			stock = append(stock, &s)
		}
		out = stock
		return nil
	})
	return out, err
}
