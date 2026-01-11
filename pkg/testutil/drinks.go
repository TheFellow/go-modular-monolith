package testutil

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
)

type DrinkBuilder struct {
	fix         *Fixture
	name        string
	ingredients []struct {
		name   string
		amount float64
	}
}

func (f *Fixture) CreateDrink(name string) *DrinkBuilder {
	f.T.Helper()
	return &DrinkBuilder{fix: f, name: name}
}

func (b *DrinkBuilder) With(ingredientName string, amount float64) *DrinkBuilder {
	b.fix.T.Helper()
	b.ingredients = append(b.ingredients, struct {
		name   string
		amount float64
	}{name: ingredientName, amount: amount})
	return b
}

func (b *DrinkBuilder) Build() *models.Drink {
	b.fix.T.Helper()

	var recipeIngredients []models.RecipeIngredient
	for _, ing := range b.ingredients {
		ingredient := b.fix.findOrCreateIngredient(ing.name)
		recipeIngredients = append(recipeIngredients, models.RecipeIngredient{
			IngredientID: ingredient.ID,
			Amount:       ing.amount,
			Unit:         ingredient.Unit,
		})
	}

	drink, err := b.fix.Drinks.Create(b.fix.OwnerContext(), models.Drink{
		Name:     b.name,
		Category: models.DrinkCategoryCocktail,
		Glass:    models.GlassTypeCoupe,
		Recipe: models.Recipe{
			Ingredients: recipeIngredients,
			Steps:       []string{"mix"},
		},
	})
	Ok(b.fix.T, err)
	return drink
}

func (f *Fixture) findOrCreateIngredient(name string) *ingredientsmodels.Ingredient {
	f.T.Helper()

	ings, err := f.Ingredients.List(f.OwnerContext(), ingredients.ListRequest{})
	Ok(f.T, err)

	want := normalizeName(name)
	for _, ing := range ings {
		if normalizeName(ing.Name) == want {
			return ing
		}
	}

	created, err := f.Ingredients.Create(f.OwnerContext(), ingredientsmodels.Ingredient{
		Name:     name,
		Category: ingredientsmodels.CategorySpirit,
		Unit:     ingredientsmodels.UnitOz,
	})
	Ok(f.T, err)
	return created
}
