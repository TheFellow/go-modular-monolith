package commands

import (
	"math"
	"time"

	ingredientauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	pkgAuthz "github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	cedar "github.com/cedar-policy/cedar-go"
)

type RetirementTarget struct {
	Ingredient *models.Ingredient
	Retirement models.Retirement
}

func (t RetirementTarget) CedarEntity() cedar.Entity { return t.Ingredient.CedarEntity() }

func (c *Commands) Retire(ctx *middleware.Context, target RetirementTarget) (*models.Ingredient, error) {
	ingredient := target.Ingredient
	if ingredient == nil {
		return nil, errors.Invalidf("ingredient is required")
	}
	if ingredient.ID.IsZero() {
		return nil, errors.Invalidf("id is required")
	}

	var replacement *models.Ingredient
	ratio := target.Retirement.Ratio
	if target.Retirement.HasReplacement() {
		if target.Retirement.ReplacementID == ingredient.ID {
			return nil, errors.Invalidf("replacement ingredient must differ from retired ingredient")
		}
		var err error
		replacement, err = c.dao.Get(ctx, target.Retirement.ReplacementID)
		if err != nil {
			return nil, errors.Invalidf("replacement ingredient %s must exist and be active: %w", target.Retirement.ReplacementID.String(), err)
		}
		if err := pkgAuthz.AuthorizeWithEntity(ctx.Principal(), ingredientauthz.ActionGet, replacement.CedarEntity()); err != nil {
			return nil, err
		}
		if replacement.Category != ingredient.Category {
			return nil, errors.Invalidf("replacement category %q is incompatible with retired category %q", replacement.Category, ingredient.Category)
		}
		if ratio == 0 {
			ratio = 1
		}
		if ratio <= 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
			return nil, errors.Invalidf("replacement ratio must be a finite number greater than zero")
		}
		amount, err := measurement.NewAmount(1, ingredient.Unit)
		if err != nil {
			return nil, errors.Internalf("retired ingredient has invalid unit %q: %w", ingredient.Unit, err)
		}
		if _, err := amount.Convert(replacement.Unit); err != nil {
			return nil, errors.Invalidf("replacement unit %q is incompatible with retired unit %q: %w", replacement.Unit, ingredient.Unit, err)
		}
	} else if ratio != 0 {
		return nil, errors.Invalidf("replacement ratio requires a replacement ingredient")
	}

	now := time.Now().UTC()
	deleted := *ingredient
	deleted.DeletedAt = optional.Some(now)

	if err := c.dao.Update(ctx, &deleted); err != nil {
		return nil, err
	}

	ctx.TouchEntity(deleted.ID.EntityUID())
	if replacement != nil {
		ctx.TouchEntity(replacement.ID.EntityUID())
	}
	ctx.AddEvent(events.IngredientDeleted{
		Ingredient:       deleted,
		DeletedAt:        now,
		Replacement:      replacement,
		ReplacementRatio: ratio,
	})

	return &deleted, nil
}

func (c *Commands) Delete(ctx *middleware.Context, ingredient *models.Ingredient) (*models.Ingredient, error) {
	return c.Retire(ctx, RetirementTarget{Ingredient: ingredient})
}
