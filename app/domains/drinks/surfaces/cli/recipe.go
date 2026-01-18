package cli

import (
	"encoding/json"
	"io"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
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
				IngredientID: "ing-abc123",
				Amount:       1.0,
				Unit:         string(ingredientsmodels.UnitOz),
				Substitutes:  []string{"ing-def456"},
			},
			{
				IngredientID: "ing-ghi789",
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
		subUIDs := make([]entity.IngredientID, 0, len(ing.Substitutes))
		for _, sub := range ing.Substitutes {
			subID, err := entity.ParseIngredientID(sub)
			if err != nil {
				return models.Recipe{}, errors.Invalidf("invalid substitute ingredient id %q: %w", sub, err)
			}
			subUIDs = append(subUIDs, subID)
		}

		ingredientID, err := entity.ParseIngredientID(ing.IngredientID)
		if err != nil {
			return models.Recipe{}, errors.Invalidf("invalid ingredient id %q: %w", ing.IngredientID, err)
		}
		out.Ingredients = append(out.Ingredients, models.RecipeIngredient{
			IngredientID: ingredientID,
			Amount:       ing.Amount,
			Unit:         measurement.Unit(ing.Unit),
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
			subs = append(subs, sub.String())
		}

		out.Ingredients = append(out.Ingredients, RecipeIngredient{
			IngredientID: ing.IngredientID.String(),
			Amount:       ing.Amount,
			Unit:         string(ing.Unit),
			Optional:     ing.Optional,
			Substitutes:  subs,
		})
	}

	return out
}
