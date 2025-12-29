package dao

import (
	"time"

	domain "github.com/TheFellow/go-modular-monolith/app/drinks/models"
)

type Drink struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func (d Drink) ToDomain() domain.Drink {
	return domain.Drink{
		ID:   d.ID,
		Name: d.Name,
	}
}

func FromDomain(d domain.Drink) Drink {
	return Drink{
		ID:   d.ID,
		Name: d.Name,
	}
}
