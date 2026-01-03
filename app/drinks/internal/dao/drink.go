package dao

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	cedar "github.com/cedar-policy/cedar-go"
)

type Drink struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Category    string     `json:"category,omitempty"`
	Glass       string     `json:"glass,omitempty"`
	Recipe      Recipe     `json:"recipe,omitempty"`
	Description string     `json:"description,omitempty"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type Recipe struct {
	Ingredients []RecipeIngredient `json:"ingredients,omitempty"`
	Steps       []string           `json:"steps,omitempty"`
	Garnish     string             `json:"garnish,omitempty"`
}

type RecipeIngredient struct {
	IngredientID string   `json:"ingredient_id"`
	Amount       float64  `json:"amount"`
	Unit         string   `json:"unit"`
	Optional     bool     `json:"optional,omitempty"`
	Substitutes  []string `json:"substitutes,omitempty"`
}

func (d Drink) ToDomain() models.Drink {
	var recipeIngredients []models.RecipeIngredient
	for _, ing := range d.Recipe.Ingredients {
		var subs []cedar.EntityUID
		for _, sub := range ing.Substitutes {
			subs = append(subs, ingredientsmodels.NewIngredientID(sub))
		}

		recipeIngredients = append(recipeIngredients, models.RecipeIngredient{
			IngredientID: ingredientsmodels.NewIngredientID(ing.IngredientID),
			Amount:       ing.Amount,
			Unit:         ingredientsmodels.Unit(ing.Unit),
			Optional:     ing.Optional,
			Substitutes:  subs,
		})
	}

	return models.Drink{
		ID:          cedar.NewEntityUID(models.DrinkEntityType, cedar.String(d.ID)),
		Name:        d.Name,
		Category:    models.DrinkCategory(d.Category),
		Glass:       models.GlassType(d.Glass),
		Recipe:      models.Recipe{Ingredients: recipeIngredients, Steps: d.Recipe.Steps, Garnish: d.Recipe.Garnish},
		Description: d.Description,
	}
}

func FromDomain(d models.Drink) Drink {
	recipeIngredients := make([]RecipeIngredient, 0, len(d.Recipe.Ingredients))
	for _, ing := range d.Recipe.Ingredients {
		subs := make([]string, 0, len(ing.Substitutes))
		for _, sub := range ing.Substitutes {
			subs = append(subs, string(sub.ID))
		}

		recipeIngredients = append(recipeIngredients, RecipeIngredient{
			IngredientID: string(ing.IngredientID.ID),
			Amount:       ing.Amount,
			Unit:         string(ing.Unit),
			Optional:     ing.Optional,
			Substitutes:  subs,
		})
	}

	return Drink{
		ID:          string(d.ID.ID),
		Name:        d.Name,
		Category:    string(d.Category),
		Glass:       string(d.Glass),
		Recipe:      Recipe{Ingredients: recipeIngredients, Steps: d.Recipe.Steps, Garnish: d.Recipe.Garnish},
		Description: d.Description,
	}
}

func (d Drink) EntityUID() cedar.EntityUID {
	return models.NewDrinkID(d.ID)
}
