package queries

import (
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

// IngredientFulfillment is the selected stock source for one required recipe
// ingredient. It is shared by menu readiness and order completion so a drink
// that is advertised as available is fulfilled by the same deterministic
// original/substitute choice.
type IngredientFulfillment struct {
	IngredientID     entity.IngredientID
	Required         measurement.Amount
	Available        measurement.Amount
	UsedSubstitution bool
	Ratio            float64
	QualityImpact    ingredientsmodels.Quality
}

// FulfillIngredient selects an in-stock original or substitute using the menu
// availability policy. False means no candidate can satisfy the requirement.
func (q *Queries) FulfillIngredient(ctx store.Context, req drinksmodels.RecipeIngredient) (IngredientFulfillment, bool) {
	pick, ok := q.availability.PickIngredient(ctx, req)
	if !ok {
		return IngredientFulfillment{}, false
	}
	return IngredientFulfillment{
		IngredientID:     pick.IngredientID,
		Required:         pick.Required,
		Available:        pick.Available,
		UsedSubstitution: pick.UsedSubstitution,
		Ratio:            pick.Ratio,
		QualityImpact:    pick.QualityImpact,
	}, true
}
