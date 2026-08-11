package dao

import (
	"time"

	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func toRow(o models.Order) OrderRow {
	var completedAt *time.Time
	if t, ok := o.CompletedAt.Unwrap(); ok {
		completedAt = &t
	}
	var deletedAt *time.Time
	if t, ok := o.DeletedAt.Unwrap(); ok {
		deletedAt = &t
	}
	items := make([]OrderItemRow, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, OrderItemRow{
			DrinkID:  it.DrinkID.EntityUID(),
			Quantity: it.Quantity,
			Notes:    it.Notes,
		})
	}
	usage := make([]IngredientUsageRow, 0, len(o.IngredientUsage))
	for _, u := range o.IngredientUsage {
		usage = append(usage, IngredientUsageRow{IngredientID: u.IngredientID.String(), Name: u.Name, Quantity: u.Amount.Value(), Unit: string(u.Amount.Unit())})
	}

	blockedIngredients := make([]string, 0, len(o.BlockedIngredients))
	for _, id := range o.BlockedIngredients {
		blockedIngredients = append(blockedIngredients, id.String())
	}
	return OrderRow{
		ID:                 o.ID.String(),
		Revision:           o.Revision,
		MenuID:             o.MenuID.String(),
		Items:              items,
		IngredientUsage:    usage,
		BlockedIngredients: blockedIngredients,
		Status:             string(o.Status),
		CreatedAt:          o.CreatedAt,
		CompletedAt:        completedAt,
		Notes:              o.Notes,
		DeletedAt:          deletedAt,
	}
}

func toModel(r OrderRow) models.Order {
	var completedAt optional.Value[time.Time]
	if r.CompletedAt != nil {
		completedAt = optional.Some(*r.CompletedAt)
	} else {
		completedAt = optional.None[time.Time]()
	}
	var deletedAt optional.Value[time.Time]
	if r.DeletedAt != nil {
		deletedAt = optional.Some(*r.DeletedAt)
	} else {
		deletedAt = optional.None[time.Time]()
	}
	items := make([]models.OrderItem, 0, len(r.Items))
	for _, it := range r.Items {
		items = append(items, models.OrderItem{
			DrinkID:  entity.DrinkID(it.DrinkID),
			Quantity: it.Quantity,
			Notes:    it.Notes,
		})
	}
	usage := make([]models.IngredientUsage, 0, len(r.IngredientUsage))
	for _, u := range r.IngredientUsage {
		ingredientID, err := entity.ParseIngredientID(u.IngredientID)
		if err != nil {
			panic(err)
		}
		usage = append(usage, models.IngredientUsage{IngredientID: ingredientID, Name: u.Name, Amount: measurement.MustAmount(u.Quantity, measurement.Unit(u.Unit))})
	}
	var blocked []entity.IngredientID
	if len(r.BlockedIngredients) > 0 {
		blocked = make([]entity.IngredientID, 0, len(r.BlockedIngredients))
	}
	for _, id := range r.BlockedIngredients {
		parsed, err := entity.ParseIngredientID(id)
		if err != nil {
			panic(err)
		}
		blocked = append(blocked, parsed)
	}

	return models.Order{
		ID:                 models.NewOrderID(r.ID),
		Revision:           r.Revision,
		MenuID:             menumodels.NewMenuID(r.MenuID),
		Items:              items,
		IngredientUsage:    usage,
		BlockedIngredients: blocked,
		Status:             models.OrderStatus(r.Status),
		CreatedAt:          r.CreatedAt,
		CompletedAt:        completedAt,
		Notes:              r.Notes,
		DeletedAt:          deletedAt,
	}
}
