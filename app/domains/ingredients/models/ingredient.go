package models

import (
	"slices"
	"strings"
	"time"

	ingredientauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	cedar "github.com/cedar-policy/cedar-go"
)

const IngredientEntityType = entity.TypeIngredient

type Ingredient struct {
	ID          entity.IngredientID
	Revision    uint64 `json:"revision"`
	Name        string
	Category    Category
	Unit        measurement.Unit
	Description string
	DeletedAt   optional.Value[time.Time]
	Tags        tag.Tags
}

// Retirement describes a deliberate ingredient lifecycle transition. A
// permanent replacement is explicit product intent; temporary substitutions
// are fulfillment options and are never promoted by this operation.
type Retirement struct {
	ReplacementID entity.IngredientID
	Ratio         float64
}

func (r Retirement) HasReplacement() bool { return !r.ReplacementID.IsZero() }

func (i Ingredient) EntityUID() cedar.EntityUID {
	return i.ID.EntityUID()
}

func (i *Ingredient) SetTags(tags tag.Tags) { i.Tags = tags }

func (i Ingredient) CedarEntity() cedar.Entity {
	return ingredientauthz.Ingredient{
		UID: i.ID.EntityUID(), Name: i.Name, Category: string(i.Category), Unit: string(i.Unit), Tags: i.Tags.Map(),
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
