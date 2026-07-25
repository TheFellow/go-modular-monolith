package dao

import (
	"iter"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	appfilter "github.com/TheFellow/go-modular-monolith/pkg/filter"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

// ListFilter specifies optional filters for listing stock rows.
type ListFilter struct {
	IngredientID entity.IngredientID
	MaxQuantity  optional.Value[float64]
	MinQuantity  optional.Value[float64]
	BeforeID     string
	Expression   *appfilter.Expression[models.ListFilterView]
}

func (d *DAO) List(ctx store.Context, filter ListFilter) iter.Seq2[*models.Inventory, error] {
	return func(yield func(*models.Inventory, error) bool) {
		err := d.store.ReadContext(ctx, func(tx *bstore.Tx) error {
			rows, err := d.query(tx, filter).SortDesc("InventoryID").List()
			if err != nil {
				return store.MapError(err, "list stock")
			}
			ids := make([]cedar.String, len(rows))
			for i := range rows {
				ids[i] = cedar.String(rows[i].InventoryID)
			}
			tagsByTarget, err := d.tags.ListTypeTx(tx, entity.TypeInventory, ids)
			if err != nil {
				return err
			}
			for _, row := range rows {
				stock := toModel(row)
				stock.Tags = tagsByTarget[stock.EntityUID()]
				if !yield(&stock, nil) {
					return nil
				}
			}
			return nil
		})
		if err != nil {
			yield(nil, err)
		}
	}
}

func (d *DAO) query(tx *bstore.Tx, filter ListFilter) *bstore.Query[StockRow] {
	q := bstore.QueryTx[StockRow](tx)

	if !filter.IngredientID.IsZero() {
		q = q.FilterID(filter.IngredientID.String())
	}
	if v, ok := filter.MaxQuantity.Unwrap(); ok {
		q = q.FilterLessEqual("Quantity", v)
	}
	if v, ok := filter.MinQuantity.Unwrap(); ok {
		q = q.FilterGreaterEqual("Quantity", v)
	}
	if filter.BeforeID != "" {
		q = q.FilterLess("InventoryID", filter.BeforeID)
	}
	q = appfilter.ApplyBstore(q, filter.Expression, func(r StockRow) models.ListFilterView {
		return models.ListFilterView{ID: r.InventoryID, IngredientID: r.IngredientID, Quantity: r.Quantity, Unit: r.Unit, LastUpdated: r.LastUpdated}
	})

	return q
}
