package drinks

import "github.com/TheFellow/go-modular-monolith/app/drinks/models"

type CreateRequest struct {
	Name string
}

type CreateResponse struct {
	Drink models.Drink
}
