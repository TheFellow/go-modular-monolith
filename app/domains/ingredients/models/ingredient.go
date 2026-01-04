package models

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/cedar-policy/cedar-go"
)

const IngredientEntityType = cedar.EntityType("Mixology::Ingredient")

func NewIngredientID(id string) cedar.EntityUID {
	return cedar.NewEntityUID(IngredientEntityType, cedar.String(id))
}

type Ingredient struct {
	ID          string
	Name        string   `bstore:"unique"`
	Category    Category `bstore:"index"`
	Unit        Unit
	Description string
}

func (i Ingredient) EntityUID() cedar.EntityUID {
	return NewIngredientID(i.ID)
}

func (i Ingredient) CedarEntity() cedar.Entity {
	uid := NewIngredientID(i.ID)
	return cedar.Entity{
		UID:        uid,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
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
	for _, v := range AllCategories() {
		if c == v {
			return nil
		}
	}
	return errors.Invalidf("invalid category %q", string(c))
}

type Unit string

const (
	UnitOz     Unit = "oz"
	UnitMl     Unit = "ml"
	UnitDash   Unit = "dash"
	UnitPiece  Unit = "piece"
	UnitSplash Unit = "splash"
)

func AllUnits() []Unit {
	return []Unit{
		UnitOz,
		UnitMl,
		UnitDash,
		UnitPiece,
		UnitSplash,
	}
}

func (u Unit) Validate() error {
	u = Unit(strings.TrimSpace(string(u)))
	if u == "" {
		return errors.Invalidf("unit is required")
	}
	for _, v := range AllUnits() {
		if u == v {
			return nil
		}
	}
	return errors.Invalidf("invalid unit %q", string(u))
}
