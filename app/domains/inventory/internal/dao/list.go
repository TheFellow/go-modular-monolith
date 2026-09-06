package dao

import (
	"iter"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	appfilter "github.com/TheFellow/go-modular-monolith/pkg/filter"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
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
		err := d.store.ReadContext(ctx, func(tx *store.Tx) error {
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
				stock, err := toModel(row)
				if err != nil {
					return err
				}
				if v, ok := filter.MaxQuantity.Unwrap(); ok && stock.Amount.Value() > v {
					continue
				}
				if v, ok := filter.MinQuantity.Unwrap(); ok && stock.Amount.Value() < v {
					continue
				}
				reserved, err := reservedQuantityTx(tx, row.IngredientID, measurement.Unit(row.Unit))
				if err != nil {
					return err
				}
				if reserved > 0 {
					stock.Reserved, err = measurement.MustAmount(reserved, measurement.Unit(row.Unit)).Convert(stock.Amount.Unit())
					if err != nil {
						return err
					}
				}
				stock.Tags = tagsByTarget[stock.EntityUID()]
				matched, err := filter.Expression.Match(listFilterView(stock, stock.Tags.Strings()))
				if err != nil {
					return err
				}
				if !matched {
					continue
				}
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

func (d *DAO) query(tx *store.Tx, filter ListFilter) *store.Query[StockRow] {
	q := store.QueryTx[StockRow](tx)

	if !filter.IngredientID.IsZero() {
		q = q.FilterID(filter.IngredientID.String())
	}
	if filter.BeforeID != "" {
		q = q.FilterLess("InventoryID", filter.BeforeID)
	}
	// Quantity and unit filters operate on display values, not canonical storage.

	return q
}

func listFilterView(value models.Inventory, tags []string) models.ListFilterView {
	return models.ListFilterView{ID: value.ID.String(), IngredientID: value.IngredientID.String(), Quantity: value.Amount.Value(), Unit: string(value.Amount.Unit()), LastUpdated: value.LastUpdated, Tags: tags}
}
