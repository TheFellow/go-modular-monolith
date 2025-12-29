package drinks

import "github.com/TheFellow/go-modular-monolith/app/drinks/models"

type GetRequest struct {
	ID string
}

type GetResponse struct {
	Drink models.Drink
}
