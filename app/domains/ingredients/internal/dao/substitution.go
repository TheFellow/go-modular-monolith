package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type SubstitutionRow struct {
	ID           string
	Revision     uint64 `json:"-" store:"revision"`
	IngredientID string `store:"index"`
	Rule         models.SubstitutionRule
}

func (d *DAO) SetSubstitution(ctx store.Context, rule *models.SubstitutionRule) error {
	return store.Write(ctx, func(tx *store.Tx) error {
		row := SubstitutionRow{ID: rule.IngredientID.String() + ":" + rule.SubstituteID.String(), Revision: rule.Revision, IngredientID: rule.IngredientID.String(), Rule: *rule}
		var err error
		if row.Revision == 0 {
			err = tx.Insert(&row)
		} else {
			err = tx.Update(&row)
		}
		if err != nil {
			return store.MapError(err, "save substitution")
		}
		rule.Revision = row.Revision
		return nil
	})
}
func (d *DAO) SubstitutionsFor(ctx store.Context, id entity.IngredientID, includeDisabled ...bool) ([]models.SubstitutionRule, error) {
	var result []models.SubstitutionRule
	err := d.store.ReadContext(ctx, func(tx *store.Tx) error {
		rows, err := store.QueryTx[SubstitutionRow](tx).FilterEqual("IngredientID", id.String()).SortAsc("ID").List()
		if err != nil {
			return err
		}
		for _, row := range rows {
			if !row.Rule.Disabled || (len(includeDisabled) > 0 && includeDisabled[0]) {
				rule := row.Rule
				rule.Revision = row.Revision
				result = append(result, rule)
			}
		}
		return nil
	})
	return result, err
}
