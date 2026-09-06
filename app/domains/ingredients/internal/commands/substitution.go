package commands

import (
	ingredientauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	pkgAuthz "github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) SetSubstitution(ctx *middleware.Context, original *models.Ingredient, rule *models.SubstitutionRule) (*models.Ingredient, error) {
	if err := rule.Validate(); err != nil {
		return nil, err
	}
	if rule.IngredientID == rule.SubstituteID {
		return nil, errors.Invalidf("substitute must differ from original")
	}
	prior, err := c.dao.SubstitutionsFor(ctx, original.ID, true)
	if err != nil {
		return nil, err
	}
	var before *models.SubstitutionRule
	for _, existing := range prior {
		if existing.SubstituteID == rule.SubstituteID {
			before = &existing
			break
		}
	}
	substitute, err := c.dao.Get(ctx, rule.SubstituteID)
	if err != nil && (!rule.Disabled || !errors.IsNotFound(err)) {
		return nil, err
	}
	if substitute != nil {
		if err := pkgAuthz.AuthorizeWithEntity(ctx.Principal(), ingredientauthz.ActionGet, substitute.CedarEntity()); err != nil {
			return nil, err
		}
		if _, err := measurement.MustAmount(1, original.Unit).Convert(substitute.Unit); err != nil {
			return nil, err
		}
	}
	if err := c.dao.SetSubstitution(ctx, rule); err != nil {
		return nil, err
	}
	ctx.RecordEffect("substitution_rule_changed", original.ID.EntityUID(), middleware.Change("rule", before, *rule))
	ctx.ReferenceEntity(rule.SubstituteID.EntityUID())
	ctx.AddEvent(events.IngredientUpdated{Ingredient: *original})
	return original, nil
}
