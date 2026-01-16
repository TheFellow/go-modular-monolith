package models

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/cedar-policy/cedar-go"
)

const DrinkEntityType = entity.TypeDrink

func NewDrinkID(id string) cedar.EntityUID {
	return entity.DrinkID(id)
}

type Drink struct {
	ID          cedar.EntityUID
	Name        string
	Category    DrinkCategory
	Glass       GlassType
	Recipe      Recipe
	Description string
	DeletedAt   optional.Value[time.Time]
}

func (d Drink) EntityUID() cedar.EntityUID {
	return d.ID
}

func (d Drink) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:     d.ID,
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
