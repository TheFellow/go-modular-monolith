package dao

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	cedar "github.com/cedar-policy/cedar-go"
)

type Drink struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func (d Drink) ToDomain() models.Drink {
	return models.Drink{
		ID:   cedar.NewEntityUID(models.DrinkEntityType, cedar.String(d.ID)),
		Name: d.Name,
	}
}

func FromDomain(d models.Drink) Drink {
	return Drink{
		ID:   string(d.ID.ID),
		Name: d.Name,
	}
}

func (d Drink) EntityUID() cedar.EntityUID {
	return models.NewDrinkID(d.ID)
}
