package dao

import (
	"time"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	cedar "github.com/cedar-policy/cedar-go"
)

func toRow(d drinksmodels.Drink) DrinkRow {
	var deletedAt *time.Time
	if t, ok := d.DeletedAt.Unwrap(); ok {
		deletedAt = &t
	}
	return DrinkRow{
		ID:          d.ID.String(),
		Name:        d.Name,
		Category:    string(d.Category),
		Glass:       string(d.Glass),
		Recipe:      toRecipeRow(d.Recipe),
		Description: d.Description,
		DeletedAt:   deletedAt,
	}
}

func toModel(r DrinkRow) drinksmodels.Drink {
	var deletedAt optional.Value[time.Time]
	if r.DeletedAt != nil {
		deletedAt = optional.Some(*r.DeletedAt)
	} else {
		deletedAt = optional.None[time.Time]()
	}
	return drinksmodels.Drink{
		ID:          drinksmodels.NewDrinkID(r.ID),
		Name:        r.Name,
		Category:    drinksmodels.DrinkCategory(r.Category),
		Glass:       drinksmodels.GlassType(r.Glass),
		Recipe:      toRecipeModel(r.Recipe),
		Description: r.Description,
		DeletedAt:   deletedAt,
	}
}

func toRecipeRow(r drinksmodels.Recipe) RecipeRow {
	var ingredients []RecipeIngredientRow
	if len(r.Ingredients) > 0 {
		ingredients = make([]RecipeIngredientRow, 0, len(r.Ingredients))
	}
	for _, ri := range r.Ingredients {
		subs := make([]cedar.EntityUID, 0, len(ri.Substitutes))
		for _, sub := range ri.Substitutes {
			subs = append(subs, sub.EntityUID())
		}
		ingredients = append(ingredients, RecipeIngredientRow{
			IngredientID: ri.IngredientID.EntityUID(),
			Amount:       ri.Amount,
			Unit:         string(ri.Unit),
			Optional:     ri.Optional,
			Substitutes:  subs,
		})
	}
	return RecipeRow{
		Ingredients: ingredients,
		Steps:       r.Steps,
		Garnish:     r.Garnish,
	}
}

func toRecipeModel(r RecipeRow) drinksmodels.Recipe {
	var ingredients []drinksmodels.RecipeIngredient
	if len(r.Ingredients) > 0 {
		ingredients = make([]drinksmodels.RecipeIngredient, 0, len(r.Ingredients))
	}
	for _, ri := range r.Ingredients {
		substitutes := make([]entity.IngredientID, 0, len(ri.Substitutes))
		for _, sub := range ri.Substitutes {
			substitutes = append(substitutes, entity.IngredientID(sub))
		}
		ingredients = append(ingredients, drinksmodels.RecipeIngredient{
			IngredientID: entity.IngredientID(ri.IngredientID),
			Amount:       ri.Amount,
			Unit:         measurement.Unit(ri.Unit),
			Optional:     ri.Optional,
			Substitutes:  substitutes,
		})
	}
	return drinksmodels.Recipe{
		Ingredients: ingredients,
		Steps:       r.Steps,
		Garnish:     r.Garnish,
	}
}
