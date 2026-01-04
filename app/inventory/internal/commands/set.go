package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type SetParams struct {
	IngredientID cedar.EntityUID
	Quantity     float64
}

func (p SetParams) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:        models.NewStockID(p.IngredientID),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}

func (c *Commands) Set(ctx *middleware.Context, params SetParams) (models.Stock, error) {
	if string(params.IngredientID.ID) == "" {
		return models.Stock{}, errors.Invalidf("ingredient id is required")
	}
	if params.Quantity < 0 {
		params.Quantity = 0
	}
	if c.ingredients == nil {
		return models.Stock{}, errors.Internalf("missing ingredients dependency")
	}

	ingredient, err := c.ingredients.Get(ctx, params.IngredientID)
	if err != nil {
		return models.Stock{}, err
	}
	if ingredient.Unit == "" {
		return models.Stock{}, errors.Invalidf("ingredient unit is required")
	}

	tx, ok := ctx.UnitOfWork()
	if !ok {
		return models.Stock{}, errors.Internalf("missing unit of work")
	}
	if err := tx.Register(c.dao); err != nil {
		return models.Stock{}, errors.Internalf("register dao: %w", err)
	}

	ingredientIDStr := string(params.IngredientID.ID)
	existing, found, err := c.dao.Get(ctx, ingredientIDStr)
	if err != nil {
		return models.Stock{}, errors.Internalf("get stock %s: %w", ingredientIDStr, err)
	}
	if !found {
		existing = dao.Stock{
			IngredientID: ingredientIDStr,
			Quantity:     0,
			Unit:         string(ingredient.Unit),
			LastUpdated:  time.Time{},
		}
	}

	previousQty := existing.Quantity
	newQty := params.Quantity
	delta := newQty - previousQty

	existing.Quantity = newQty
	existing.Unit = string(ingredient.Unit)
	existing.LastUpdated = time.Now().UTC()

	if err := c.dao.Set(ctx, existing); err != nil {
		return models.Stock{}, err
	}

	ctx.AddEvent(events.StockAdjusted{
		IngredientID: params.IngredientID,
		PreviousQty:  previousQty,
		NewQty:       newQty,
		Delta:        delta,
		Reason:       "set",
	})

	return existing.ToDomain(), nil
}
