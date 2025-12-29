package drinks

import "github.com/TheFellow/go-modular-monolith/app/drinks/models"

type ListRequest struct{}

type ListResponse struct {
	Drinks []models.Drink
}
