package dao

import (
	"time"

	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	cedar "github.com/cedar-policy/cedar-go"
)

type Stock struct {
	IngredientID string       `json:"ingredient_id"`
	Quantity     float64      `json:"quantity"`
	Unit         string       `json:"unit"`
	CostPerUnit  *money.Price `json:"cost_per_unit,omitempty"`
	LastUpdated  time.Time    `json:"last_updated"`
}

func (s Stock) ToDomain() models.Stock {
	var cost = optional.None[money.Price]()
	if s.CostPerUnit != nil {
		cost = optional.Some(*s.CostPerUnit)
	}
	return models.Stock{
		IngredientID: ingredientsmodels.NewIngredientID(s.IngredientID),
		Quantity:     s.Quantity,
		Unit:         ingredientsmodels.Unit(s.Unit),
		CostPerUnit:  cost,
		LastUpdated:  s.LastUpdated,
	}
}

func FromDomain(s models.Stock) Stock {
	var cost *money.Price
	if v, ok := s.CostPerUnit.Unwrap(); ok {
		cost = &v
	}
	return Stock{
		IngredientID: string(s.IngredientID.ID),
		Quantity:     s.Quantity,
		Unit:         string(s.Unit),
		CostPerUnit:  cost,
		LastUpdated:  s.LastUpdated,
	}
}

func (s Stock) EntityUID() cedar.EntityUID {
	return cedar.NewEntityUID(models.StockEntityType, cedar.String(s.IngredientID))
}
