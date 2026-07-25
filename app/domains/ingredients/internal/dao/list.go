package dao

import (
	"iter"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	appfilter "github.com/TheFellow/go-modular-monolith/pkg/filter"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

// ListFilter specifies optional filters for listing ingredients.
type ListFilter struct {
	Category models.Category
	Name     string // Exact match on Name (uses bstore unique index)
	IDs      []entity.IngredientID
	// IncludeDeleted includes soft-deleted rows (DeletedAt != nil).
	IncludeDeleted bool
	BeforeID       string
	Expression     *appfilter.Expression[models.ListFilterView]
}

func (d *DAO) List(ctx store.Context, filter ListFilter) iter.Seq2[*models.Ingredient, error] {
	return func(yield func(*models.Ingredient, error) bool) {
		err := d.store.ReadContext(ctx, func(tx *bstore.Tx) error {
			rows, err := d.query(tx, filter).SortDesc("ID").List()
			if err != nil {
				return store.MapError(err, "list ingredients")
			}
			ids := make([]cedar.String, len(rows))
			for i := range rows {
				ids[i] = cedar.String(rows[i].ID)
			}
			tagsByTarget, err := d.tags.ListTypeTx(tx, entity.TypeIngredient, ids)
			if err != nil {
				return err
			}
			for _, row := range rows {
				ingredient := toModel(row)
				ingredient.Tags = tagsByTarget[ingredient.EntityUID()]
				matched, err := filter.Expression.Match(listFilterView(row, ingredient.Tags.Strings()))
				if err != nil {
					return err
				}
				if !matched {
					continue
				}
				if !yield(&ingredient, nil) {
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

func (d *DAO) query(tx *bstore.Tx, filter ListFilter) *bstore.Query[IngredientRow] {
	q := bstore.QueryTx[IngredientRow](tx)
	if filter.Category != "" {
		q = q.FilterEqual("Category", string(filter.Category))
	}
	if filter.Name != "" {
		q = q.FilterEqual("Name", filter.Name)
	}
	if len(filter.IDs) > 0 {
		idSet := make(map[string]struct{}, len(filter.IDs))
		for _, id := range filter.IDs {
			idSet[id.String()] = struct{}{}
		}
		q = q.FilterFn(func(r IngredientRow) bool {
			_, ok := idSet[r.ID]
			return ok
		})
	}
	if filter.BeforeID != "" {
		q = q.FilterLess("ID", filter.BeforeID)
	}
	if !filter.IncludeDeleted {
		q = q.FilterFn(func(r IngredientRow) bool {
			return r.DeletedAt == nil
		})
	}
	q = appfilter.ApplyBstorePushdowns(q, filter.Expression)
	return q
}

func listFilterView(r IngredientRow, tags []string) models.ListFilterView {
	return models.ListFilterView{ID: r.ID, Name: r.Name, Category: r.Category, Unit: r.Unit, Description: r.Description, Tags: tags}
}
