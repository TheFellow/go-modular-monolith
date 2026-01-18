package models

import (
	"strings"
	"time"

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
	Unit        Unit
	Description string
	DeletedAt   optional.Value[time.Time]
}

func (i Ingredient) EntityUID() cedar.EntityUID {
	return i.ID.EntityUID()
}

func (i Ingredient) CedarEntity() cedar.Entity {
	uid := i.ID.EntityUID()
	if uid.Type == "" {
		uid = cedar.NewEntityUID(cedar.EntityType(IngredientEntityType), uid.ID)
	}
	return cedar.Entity{
		UID:     uid,
		Parents: cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(cedar.RecordMap{
			"Name":     cedar.String(i.Name),
			"Category": cedar.String(i.Category),
			"Unit":     cedar.String(i.Unit),
		}),
		Tags: cedar.NewRecord(nil),
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

type Unit = measurement.Unit

const (
	UnitOz     = measurement.UnitOz
	UnitMl     = measurement.UnitMl
	UnitDash   = measurement.UnitDash
	UnitPiece  = measurement.UnitPiece
	UnitSplash = measurement.UnitSplash
)

func AllUnits() []Unit { return measurement.AllUnits() }
