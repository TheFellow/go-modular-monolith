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

type Adjust struct {
	dao *dao.FileStockDAO
}

func NewAdjust(dao *dao.FileStockDAO) *Adjust {
	return &Adjust{dao: dao}
}

type AdjustRequest struct {
	IngredientID cedar.EntityUID
	Delta        float64
	Unit         ingredientsmodels.Unit
	Reason       models.AdjustmentReason
}

func (c *Adjust) Execute(ctx *middleware.Context, req AdjustRequest) (models.Stock, error) {
	if string(req.IngredientID.ID) == "" {
		return models.Stock{}, errors.Invalidf("ingredient id is required")
	}
	if req.Unit == "" {
		return models.Stock{}, errors.Invalidf("unit is required")
	}
	if req.Reason == "" {
		return models.Stock{}, errors.Invalidf("reason is required")
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
	newQty := previousQty + req.Delta
	if newQty < 0 {
		newQty = 0
	}
	appliedDelta := newQty - previousQty

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
		Delta:        appliedDelta,
		Reason:       string(req.Reason),
	})

	if previousQty > 0 && newQty == 0 {
		ctx.AddEvent(events.IngredientDepleted{IngredientID: req.IngredientID})
	}
	if previousQty == 0 && newQty > 0 {
		ctx.AddEvent(events.IngredientRestocked{
			IngredientID: req.IngredientID,
			NewQty:       newQty,
		})
	}

	return existing.ToDomain(), nil
}
