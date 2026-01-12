package handlers

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type IngredientCreatedCounter struct{}

func NewIngredientCreatedCounter() *IngredientCreatedCounter {
	return &IngredientCreatedCounter{}
}

func (h *IngredientCreatedCounter) Handle(_ *middleware.Context, e events.IngredientCreated) error {
	_ = e.Ingredient
	return nil
}

type IngredientCreatedAudit struct{}

func NewIngredientCreatedAudit() *IngredientCreatedAudit {
	return &IngredientCreatedAudit{}
}

func (h *IngredientCreatedAudit) Handle(_ *middleware.Context, e events.IngredientCreated) error {
	_ = e.Ingredient
	return nil
}
