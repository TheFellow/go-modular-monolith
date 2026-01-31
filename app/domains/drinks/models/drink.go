package models

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	cedar "github.com/cedar-policy/cedar-go"
)

const DrinkEntityType = entity.TypeDrink

func NewDrinkID(id string) entity.DrinkID {
	return entity.DrinkID(cedar.NewEntityUID(entity.TypeDrink, cedar.String(id)))
}

type Drink struct {
	ID          entity.DrinkID
	Name        string
	Category    DrinkCategory
	Glass       GlassType
	Recipe      Recipe
	Description string
	DeletedAt   optional.Value[time.Time]
}

func (d Drink) EntityUID() cedar.EntityUID {
	return d.ID.EntityUID()
}

func (d Drink) CedarEntity() cedar.Entity {
	uid := d.ID.EntityUID()
	if uid.Type == "" {
		uid = cedar.NewEntityUID(cedar.EntityType(DrinkEntityType), uid.ID)
	}
	return cedar.Entity{
		UID:     uid,
		Parents: cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(cedar.RecordMap{
			"Name":        cedar.String(d.Name),
			"Category":    cedar.String(d.Category),
			"Glass":       cedar.String(d.Glass),
			"Description": cedar.String(d.Description),
		}),
		Tags: cedar.NewRecord(nil),
	}
}
