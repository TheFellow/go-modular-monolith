package models

import (
	"strings"

	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

type Recipe struct {
	Ingredients []RecipeIngredient
	Steps       []string
	Garnish     string
}

type RecipeIngredient struct {
	IngredientID cedar.EntityUID
	Amount       float64
	Unit         ingredientsmodels.Unit
	Optional     bool
	Substitutes  []cedar.EntityUID
}

func (r Recipe) Validate() error {
	for i, ing := range r.Ingredients {
		if string(ing.IngredientID.ID) == "" {
			return errors.Invalidf("recipe ingredient %d: ingredient id is required", i)
		}
		if ing.Amount <= 0 {
			return errors.Invalidf("recipe ingredient %d: amount must be > 0", i)
		}
		if ing.Unit == "" {
			return errors.Invalidf("recipe ingredient %d: unit is required", i)
		}
		for j, sub := range ing.Substitutes {
			if string(sub.ID) == "" {
				return errors.Invalidf("recipe ingredient %d substitute %d: id is required", i, j)
			}
		}
	}

	for i, step := range r.Steps {
		if strings.TrimSpace(step) == "" {
			return errors.Invalidf("recipe step %d: cannot be blank", i)
		}
	}
	return nil
}

type DrinkCategory string

const (
	DrinkCategoryCocktail DrinkCategory = "cocktail"
	DrinkCategoryMocktail DrinkCategory = "mocktail"
	DrinkCategoryShot     DrinkCategory = "shot"
	DrinkCategoryHighball DrinkCategory = "highball"
	DrinkCategoryMartini  DrinkCategory = "martini"
	DrinkCategorySour     DrinkCategory = "sour"
	DrinkCategoryTiki     DrinkCategory = "tiki"
)

func (c DrinkCategory) Validate() error {
	switch c {
	case "":
		return nil
	case DrinkCategoryCocktail, DrinkCategoryMocktail, DrinkCategoryShot, DrinkCategoryHighball, DrinkCategoryMartini, DrinkCategorySour, DrinkCategoryTiki:
		return nil
	default:
		return errors.Invalidf("invalid category %q", string(c))
	}
}

type GlassType string

const (
	GlassTypeRocks    GlassType = "rocks"
	GlassTypeHighball GlassType = "highball"
	GlassTypeCoupe    GlassType = "coupe"
	GlassTypeMartini  GlassType = "martini"
)

func (g GlassType) Validate() error {
	switch g {
	case "":
		return nil
	case GlassTypeRocks, GlassTypeHighball, GlassTypeCoupe, GlassTypeMartini:
		return nil
	default:
		return errors.Invalidf("invalid glass %q", string(g))
	}
}
