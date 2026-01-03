package commands

import (
	"time"

	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type Set struct {
	dao *dao.FileStockDAO
}

func NewSet(dao *dao.FileStockDAO) *Set {
	return &Set{dao: dao}
}

type SetRequest struct {
	IngredientID cedar.EntityUID
	Quantity     float64
	Unit         ingredientsmodels.Unit
}

func (c *Set) Execute(ctx *middleware.Context, req SetRequest) (models.Stock, error) {
	if string(req.IngredientID.ID) == "" {
		return models.Stock{}, errors.Invalidf("ingredient id is required")
	}
	if req.Unit == "" {
		return models.Stock{}, errors.Invalidf("unit is required")
	}
	if req.Quantity < 0 {
		req.Quantity = 0
	}

	tx, ok := ctx.UnitOfWork()
	if !ok {
		return models.Stock{}, errors.Internalf("missing unit of work")
	}
	if err := tx.Register(c.dao); err != nil {
		return models.Stock{}, errors.Internalf("register dao: %w", err)
	}

	ingredientID := string(req.IngredientID.ID)
	existing, found, err := c.dao.Get(ctx, ingredientID)
	if err != nil {
		return models.Stock{}, errors.Internalf("get stock %s: %w", ingredientID, err)
	}
	if !found {
		existing = dao.Stock{
			IngredientID: ingredientID,
			Quantity:     0,
			Unit:         string(req.Unit),
			LastUpdated:  time.Time{},
		}
	}

	previousQty := existing.Quantity
	newQty := req.Quantity
	delta := newQty - previousQty

	existing.Quantity = newQty
	existing.Unit = string(req.Unit)
	existing.LastUpdated = time.Now().UTC()

	if err := c.dao.Set(ctx, existing); err != nil {
		return models.Stock{}, err
	}

	ctx.AddEvent(events.StockAdjusted{
		IngredientID: req.IngredientID,
		PreviousQty:  previousQty,
		NewQty:       newQty,
		Delta:        delta,
		Reason:       "set",
	})

	return existing.ToDomain(), nil
}
