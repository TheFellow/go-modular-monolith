package models

import (
	"time"

	drinkauthz "github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
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
	return drinkauthz.Drink{
		UID: d.ID.EntityUID(), Name: d.Name, Category: string(d.Category),
		Glass: string(d.Glass), Description: d.Description,
	}.CedarEntity()
}
