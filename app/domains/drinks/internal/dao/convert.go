package dao

import (
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	cedar "github.com/cedar-policy/cedar-go"
)

func toRow(d drinksmodels.Drink) DrinkRow {
	return DrinkRow{
		ID:          string(d.ID.ID),
		Name:        d.Name,
		Category:    string(d.Category),
		Glass:       string(d.Glass),
		Recipe:      toRecipeRow(d.Recipe),
		Description: d.Description,
	}
}

func toModel(r DrinkRow) drinksmodels.Drink {
	return drinksmodels.Drink{
		ID:          drinksmodels.NewDrinkID(r.ID),
		Name:        r.Name,
		Category:    drinksmodels.DrinkCategory(r.Category),
		Glass:       drinksmodels.GlassType(r.Glass),
		Recipe:      toRecipeModel(r.Recipe),
		Description: r.Description,
	}
}

func toRecipeRow(r drinksmodels.Recipe) RecipeRow {
	var ingredients []RecipeIngredientRow
	if len(r.Ingredients) > 0 {
		ingredients = make([]RecipeIngredientRow, 0, len(r.Ingredients))
	}
	for _, ri := range r.Ingredients {
		subs := make([]string, 0, len(ri.Substitutes))
		for _, sub := range ri.Substitutes {
			subs = append(subs, string(sub.ID))
		}
		ingredients = append(ingredients, RecipeIngredientRow{
			IngredientID: string(ri.IngredientID.ID),
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
		subs := make([]cedar.EntityUID, 0, len(ri.Substitutes))
		for _, sub := range ri.Substitutes {
			subs = append(subs, ingredientsmodels.NewIngredientID(sub))
		}
		ingredients = append(ingredients, drinksmodels.RecipeIngredient{
			IngredientID: ingredientsmodels.NewIngredientID(ri.IngredientID),
			Amount:       ri.Amount,
			Unit:         ingredientsmodels.Unit(ri.Unit),
			Optional:     ri.Optional,
			Substitutes:  subs,
		})
	}
	return drinksmodels.Recipe{
		Ingredients: ingredients,
		Steps:       r.Steps,
		Garnish:     r.Garnish,
	}
}
