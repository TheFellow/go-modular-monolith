package models

import cedar "github.com/cedar-policy/cedar-go"

const IngredientEntityType = cedar.EntityType("Mixology::Ingredient")

func NewIngredientID(id string) cedar.EntityUID {
	return cedar.NewEntityUID(IngredientEntityType, cedar.String(id))
}

type Ingredient struct {
	ID          cedar.EntityUID
	Name        string
	Category    Category
	Unit        Unit
	Description string
}

func (i Ingredient) EntityUID() cedar.EntityUID {
	return i.ID
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

type Unit string

const (
	UnitOz     Unit = "oz"
	UnitMl     Unit = "ml"
	UnitDash   Unit = "dash"
	UnitPiece  Unit = "piece"
	UnitSplash Unit = "splash"
)
