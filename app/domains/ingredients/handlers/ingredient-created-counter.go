package handlers

import (
	"sync/atomic"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

var (
	IngredientCreatedCount      atomic.Int64
	IngredientCreatedAuditCount atomic.Int64
)

type IngredientCreatedCounter struct{}

func NewIngredientCreatedCounter() *IngredientCreatedCounter {
	return &IngredientCreatedCounter{}
}

func (h *IngredientCreatedCounter) Handle(_ *middleware.Context, e events.IngredientCreated) error {
	_ = e.Ingredient
	IngredientCreatedCount.Add(1)
	return nil
}

type IngredientCreatedAudit struct{}

func NewIngredientCreatedAudit() *IngredientCreatedAudit {
	return &IngredientCreatedAudit{}
}

func (h *IngredientCreatedAudit) Handle(_ *middleware.Context, e events.IngredientCreated) error {
	_ = e.Ingredient
	IngredientCreatedAuditCount.Add(1)
	return nil
}
