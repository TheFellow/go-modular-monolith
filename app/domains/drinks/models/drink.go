package models

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/cedar-policy/cedar-go"
)

const DrinkEntityType = cedar.EntityType("Mixology::Drink")

func NewDrinkID(id string) cedar.EntityUID {
	return cedar.NewEntityUID(DrinkEntityType, cedar.String(id))
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
		UID:        d.ID,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}
