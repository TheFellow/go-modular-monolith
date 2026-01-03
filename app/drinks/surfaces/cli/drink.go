package cli

import "github.com/TheFellow/go-modular-monolith/app/drinks/models"

type Drink struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category,omitempty"`
	Glass       string `json:"glass,omitempty"`
	Description string `json:"description,omitempty"`
	Recipe      Recipe `json:"recipe"`
}

func FromDomainDrink(d models.Drink) Drink {
	return Drink{
		ID:          string(d.ID.ID),
		Name:        d.Name,
		Category:    string(d.Category),
		Glass:       string(d.Glass),
		Description: d.Description,
		Recipe:      FromDomainRecipe(d.Recipe),
	}
}
