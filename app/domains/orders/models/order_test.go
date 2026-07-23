package models_test

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestOrderValidateAcceptsWhitespacePaddedNotesWithoutNormalizing(t *testing.T) {
	t.Parallel()

	order := models.Order{
		MenuID: menumodels.NewMenuID("menu-1"),
		Status: models.OrderStatusPending,
		Notes:  "  rush ticket  ",
		Items: []models.OrderItem{
			{
				DrinkID:  drinksmodels.NewDrinkID("drink-1"),
				Quantity: 1,
				Notes:    "  no garnish  ",
			},
		},
	}

	testutil.Ok(t, order.Validate())
	testutil.Equals(t, order.Notes, "  rush ticket  ")
	testutil.Equals(t, order.Items[0].Notes, "  no garnish  ")
}

func TestOrderItemValidateAcceptsWhitespacePaddedNotesWithoutNormalizing(t *testing.T) {
	t.Parallel()

	item := models.OrderItem{
		DrinkID:  drinksmodels.NewDrinkID("drink-1"),
		Quantity: 1,
		Notes:    "  extra cold  ",
	}

	testutil.Ok(t, item.Validate())
	testutil.Equals(t, item.Notes, "  extra cold  ")
}
