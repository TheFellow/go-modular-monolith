// Package surfaces provides inventory workflows shared by interactive surfaces.
package surfaces

import (
	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	ingredientmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
)

// StockingCandidates returns active catalog ingredients without an existing stock
// record that the actor may initialize. Set still checks permission and revision
// at submission, including a concurrent first receipt by another editor.
func StockingCandidates(session *app.Session, projector inventory.ActionProjector) ([]ingredientmodels.Ingredient, error) {
	var candidates []ingredientmodels.Ingredient
	var cursor paging.Cursor
	for {
		page, err := session.Ingredients.List(session.Context(), ingredients.ListRequest{Cursor: cursor})
		if err != nil {
			return nil, err
		}
		for _, ingredient := range page.Items {
			_, err := session.Inventory.Get(session.Context(), ingredient.ID)
			if err == nil {
				continue
			}
			if !errors.IsNotFound(err) {
				return nil, err
			}
			stock := models.Inventory{IngredientID: ingredient.ID, Amount: measurement.MustAmount(0, ingredient.Unit)}
			states, err := projector.Project(session.Context(), session.Context().Principal(), &stock)
			if err != nil {
				return nil, err
			}
			for _, state := range states {
				if state.ID == inventory.ControlSet && state.Visible && state.Enabled {
					candidates = append(candidates, *ingredient)
				}
			}
		}
		if page.Next == "" {
			return candidates, nil
		}
		cursor = page.Next
	}
}
