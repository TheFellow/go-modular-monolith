package dao

import (
	"time"

	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/money"
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
	return models.Stock{
		IngredientID: ingredientsmodels.NewIngredientID(s.IngredientID),
		Quantity:     s.Quantity,
		Unit:         ingredientsmodels.Unit(s.Unit),
		CostPerUnit:  s.CostPerUnit,
		LastUpdated:  s.LastUpdated,
	}
}

func FromDomain(s models.Stock) Stock {
	return Stock{
		IngredientID: string(s.IngredientID.ID),
		Quantity:     s.Quantity,
		Unit:         string(s.Unit),
		CostPerUnit:  s.CostPerUnit,
		LastUpdated:  s.LastUpdated,
	}
}

func (s Stock) EntityUID() cedar.EntityUID {
	return cedar.NewEntityUID(models.StockEntityType, cedar.String(s.IngredientID))
}
