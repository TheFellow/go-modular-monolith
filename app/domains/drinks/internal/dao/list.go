package dao

import (
	"slices"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

// ListFilter specifies optional filters for listing drinks.
type ListFilter struct {
	Name     string               // Exact match on Name (uses bstore unique index)
	Category models.DrinkCategory // Exact match on Category (uses bstore index)
	Glass    models.GlassType     // Exact match on Glass
	// IncludeDeleted includes soft-deleted rows (DeletedAt != nil).
	IncludeDeleted bool
}

func (d *DAO) List(ctx store.Context, filter ListFilter) ([]*models.Drink, error) {
	var out []*models.Drink
	err := store.Read(ctx, func(tx *bstore.Tx) error {
		q := d.query(tx, filter)
		rows, err := q.SortAsc("Name").List()
		if err != nil {
			return store.MapError(err, "list drinks")
		}
		drinks := make([]*models.Drink, 0, len(rows))
		for _, r := range rows {
			if !filter.IncludeDeleted && r.DeletedAt != nil {
				continue
			}
			d := toModel(r)
			drinks = append(drinks, &d)
		}
		out = drinks
		return nil
	})
	return out, err
}

func (d *DAO) Count(ctx store.Context, filter ListFilter) (int, error) {
	var count int
	err := store.Read(ctx, func(tx *bstore.Tx) error {
		q := d.query(tx, filter)

		var err error
		count, err = q.Count()
		if err != nil {
			return store.MapError(err, "count drinks")
		}
		return nil
	})
	return count, err
}

func (d *DAO) ListByIngredient(ctx store.Context, ingredientID entity.IngredientID) ([]*models.Drink, error) {
	var out []*models.Drink
	err := store.Read(ctx, func(tx *bstore.Tx) error {
		target := ingredientID.EntityUID()
		rows, err := bstore.QueryTx[DrinkRow](tx).FilterFn(func(r DrinkRow) bool {
			if r.DeletedAt != nil {
				return false
			}
			for _, ri := range r.Recipe.Ingredients {
				if ri.IngredientID == target {
					return true
				}
				if slices.Contains(ri.Substitutes, target) {
					return true
				}
			}
			return false
		}).SortAsc("Name").List()
		if err != nil {
			return store.MapError(err, "list drinks by ingredient %s", ingredientID.String())
		}
		drinks := make([]*models.Drink, 0, len(rows))
		for _, r := range rows {
			d := toModel(r)
			drinks = append(drinks, &d)
		}
		out = drinks
		return nil
	})
	return out, err
}

func (d *DAO) query(tx *bstore.Tx, filter ListFilter) *bstore.Query[DrinkRow] {
	q := bstore.QueryTx[DrinkRow](tx)

	if filter.Name != "" {
		q = q.FilterEqual("Name", filter.Name)
	}
	if filter.Category != "" {
		q = q.FilterEqual("Category", string(filter.Category))
	}
	if filter.Glass != "" {
		q = q.FilterEqual("Glass", string(filter.Glass))
	}
	if !filter.IncludeDeleted {
		q = q.FilterFn(func(r DrinkRow) bool {
			return r.DeletedAt == nil
		})
	}

	return q
}
