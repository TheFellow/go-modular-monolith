package models

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
)

func TestOrderValidateAcceptsWhitespacePaddedNotesWithoutNormalizing(t *testing.T) {
	t.Parallel()

	order := Order{
		MenuID: menumodels.NewMenuID("menu-1"),
		Status: OrderStatusPending,
		Notes:  "  rush ticket  ",
		Items: []OrderItem{
			{
				DrinkID:  drinksmodels.NewDrinkID("drink-1"),
				Quantity: 1,
				Notes:    "  no garnish  ",
			},
		},
	}

	if err := order.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if order.Notes != "  rush ticket  " {
		t.Fatalf("Validate() normalized order notes to %q", order.Notes)
	}
	if order.Items[0].Notes != "  no garnish  " {
		t.Fatalf("Validate() normalized item notes to %q", order.Items[0].Notes)
	}
}

func TestOrderItemValidateAcceptsWhitespacePaddedNotesWithoutNormalizing(t *testing.T) {
	t.Parallel()

	item := OrderItem{
		DrinkID:  drinksmodels.NewDrinkID("drink-1"),
		Quantity: 1,
		Notes:    "  extra cold  ",
	}

	if err := item.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if item.Notes != "  extra cold  " {
		t.Fatalf("Validate() normalized item notes to %q", item.Notes)
	}
}
