package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

// SetSubstitution explicitly creates or updates an ID-based rule. Set Disabled
// to withdraw it; an update requires the revision returned by SubstitutionsFor.
func (m *Module) SetSubstitution(ctx *middleware.Context, rule *models.SubstitutionRule) (*models.SubstitutionRule, error) {
	if rule == nil {
		return nil, errors.Invalidf("rule is required")
	}
	saved := *rule
	_, err := m.pipeline.LoadCommand(ctx, authz.ActionUpdate,
		func(ctx *middleware.Context) (*models.Ingredient, error) {
			return m.queries.Get(ctx, rule.IngredientID)
		},
		func(ctx *middleware.Context, i *models.Ingredient) (*models.Ingredient, error) {
			return m.commands.SetSubstitution(ctx, i, &saved)
		})
	if err != nil {
		return nil, err
	}
	return &saved, nil
}
func (m *Module) SubstitutionsFor(ctx *middleware.Context, id entity.IngredientID) ([]models.SubstitutionRule, error) {
	if _, err := m.Get(ctx, id); err != nil {
		return nil, err
	}
	return m.queries.SubstitutionsFor(ctx, id)
}

// SubstitutionRules includes disabled rules for revision-aware administration.
func (m *Module) SubstitutionRules(ctx *middleware.Context, id entity.IngredientID) ([]models.SubstitutionRule, error) {
	if _, err := m.Get(ctx, id); err != nil {
		return nil, err
	}
	return m.queries.SubstitutionRules(ctx, id)
}
