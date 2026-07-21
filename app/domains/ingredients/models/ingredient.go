package models

import (
	"slices"
	"strings"
	"time"

	ingredientauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	cedar "github.com/cedar-policy/cedar-go"
)

const IngredientEntityType = entity.TypeIngredient

type Ingredient struct {
	ID          entity.IngredientID
	Name        string
	Category    Category
	Unit        measurement.Unit
	Description string
	DeletedAt   optional.Value[time.Time]
}

func (i Ingredient) EntityUID() cedar.EntityUID {
	return i.ID.EntityUID()
}

func (i Ingredient) CedarEntity() cedar.Entity {
	return ingredientauthz.Ingredient{
		UID: i.ID.EntityUID(), Name: i.Name, Category: string(i.Category), Unit: string(i.Unit),
	}.CedarEntity()
}

type Category string

const (
	CategorySpirit  Category = "spirit"
	CategoryMixer   Category = "mixer"
	CategoryGarnish Category = "garnish"
	CategoryBitter  Category = "bitter"
	CategorySyrup   Category = "syrup"
	CategoryJuice   Category = "juice"
	CategoryOther   Category = "other"
)

func AllCategories() []Category {
	return []Category{
		CategorySpirit,
		CategoryMixer,
		CategoryGarnish,
		CategoryBitter,
		CategorySyrup,
		CategoryJuice,
		CategoryOther,
	}
}

func (c Category) Validate() error {
	c = Category(strings.TrimSpace(string(c)))
	if c == "" {
		return errors.Invalidf("category is required")
	}
	if slices.Contains(AllCategories(), c) {
		return nil
	}
	return errors.Invalidf("invalid category %q", string(c))
}
