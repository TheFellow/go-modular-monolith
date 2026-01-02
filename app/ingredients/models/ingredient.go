package models

type Ingredient struct {
	ID          string
	Name        string
	Category    Category
	Unit        Unit
	Description string
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
