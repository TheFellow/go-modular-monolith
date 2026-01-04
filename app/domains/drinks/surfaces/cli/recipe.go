package cli

import (
	"encoding/json"
	"io"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

type Recipe struct {
	Ingredients []RecipeIngredient `json:"ingredients"`
	Steps       []string           `json:"steps"`
	Garnish     string             `json:"garnish,omitempty"`
}

type RecipeIngredient struct {
	IngredientID string   `json:"ingredient_id"`
	Amount       float64  `json:"amount"`
	Unit         string   `json:"unit"`
	Optional     bool     `json:"optional,omitempty"`
	Substitutes  []string `json:"substitutes,omitempty"`
}

func TemplateRecipe() Recipe {
	return Recipe{
		Ingredients: []RecipeIngredient{
			{
				IngredientID: "lime-juice",
				Amount:       1.0,
				Unit:         string(ingredientsmodels.UnitOz),
				Substitutes:  []string{"lemon-juice"},
			},
			{
				IngredientID: "tequila",
				Amount:       2.0,
				Unit:         string(ingredientsmodels.UnitOz),
			},
		},
		Steps:   []string{"Add ingredients to a shaker with ice", "Shake until chilled", "Strain into glass"},
		Garnish: "lime wheel",
	}
}

func DecodeRecipeJSON(r io.Reader) (models.Recipe, error) {
	var doc Recipe
	if err := json.NewDecoder(r).Decode(&doc); err != nil {
		return models.Recipe{}, errors.Invalidf("parse recipe json: %w", err)
	}
	return doc.ToDomain()
}

func (r Recipe) ToDomain() (models.Recipe, error) {
	out := models.Recipe{
		Ingredients: make([]models.RecipeIngredient, 0, len(r.Ingredients)),
		Steps:       r.Steps,
		Garnish:     r.Garnish,
	}

	for _, ing := range r.Ingredients {
		subUIDs := make([]cedar.EntityUID, 0, len(ing.Substitutes))
		for _, sub := range ing.Substitutes {
			subUIDs = append(subUIDs, ingredientsmodels.NewIngredientID(sub))
		}

		out.Ingredients = append(out.Ingredients, models.RecipeIngredient{
			IngredientID: ingredientsmodels.NewIngredientID(ing.IngredientID),
			Amount:       ing.Amount,
			Unit:         ingredientsmodels.Unit(ing.Unit),
			Optional:     ing.Optional,
			Substitutes:  subUIDs,
		})
	}

	if err := out.Validate(); err != nil {
		return models.Recipe{}, err
	}
	return out, nil
}

func FromDomainRecipe(r models.Recipe) Recipe {
	out := Recipe{
		Ingredients: make([]RecipeIngredient, 0, len(r.Ingredients)),
		Steps:       r.Steps,
		Garnish:     r.Garnish,
	}
	if out.Ingredients == nil {
		out.Ingredients = []RecipeIngredient{}
	}
	if out.Steps == nil {
		out.Steps = []string{}
	}

	for _, ing := range r.Ingredients {
		subs := make([]string, 0, len(ing.Substitutes))
		for _, sub := range ing.Substitutes {
			subs = append(subs, string(sub.ID))
		}

		out.Ingredients = append(out.Ingredients, RecipeIngredient{
			IngredientID: string(ing.IngredientID.ID),
			Amount:       ing.Amount,
			Unit:         string(ing.Unit),
			Optional:     ing.Optional,
			Substitutes:  subs,
		})
	}

	return out
}
