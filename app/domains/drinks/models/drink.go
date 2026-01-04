package models

import cedar "github.com/cedar-policy/cedar-go"

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
}

func (d Drink) EntityUID() cedar.EntityUID {
	return d.ID
}

func (d Drink) CedarEntity() cedar.Entity {
	uid := d.ID
	if string(uid.ID) == "" {
		uid = cedar.NewEntityUID(DrinkEntityType, cedar.String(""))
	}
	return cedar.Entity{
		UID:        uid,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}
