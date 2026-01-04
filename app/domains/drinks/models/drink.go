package models

import cedar "github.com/cedar-policy/cedar-go"

const DrinkEntityType = cedar.EntityType("Mixology::Drink")

func NewDrinkID(id string) cedar.EntityUID {
	return cedar.NewEntityUID(DrinkEntityType, cedar.String(id))
}

type Drink struct {
	ID          string
	Name        string        `bstore:"unique"`
	Category    DrinkCategory `bstore:"index"`
	Glass       GlassType
	Recipe      Recipe
	Description string
}

func (d Drink) EntityUID() cedar.EntityUID {
	return NewDrinkID(d.ID)
}

func (d Drink) CedarEntity() cedar.Entity {
	uid := NewDrinkID(d.ID)
	return cedar.Entity{
		UID:        uid,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}
