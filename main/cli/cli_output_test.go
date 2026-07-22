package main

import (
	"reflect"
	"slices"
	"testing"
	"time"

	auditmodels "github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	auditcli "github.com/TheFellow/go-modular-monolith/app/domains/audit/surfaces/cli"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	drinkscli "github.com/TheFellow/go-modular-monolith/app/domains/drinks/surfaces/cli"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	ingredientscli "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/cli"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	inventorycli "github.com/TheFellow/go-modular-monolith/app/domains/inventory/surfaces/cli"
	menusmodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	menuscli "github.com/TheFellow/go-modular-monolith/app/domains/menus/surfaces/cli"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	orderscli "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces/cli"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func TestCommandNouns(t *testing.T) {
	t.Parallel()

	c, err := NewCLI()
	if err != nil {
		t.Fatalf("NewCLI: %v", err)
	}

	commands := c.Command().Commands
	names := make([]string, 0, len(commands))
	for _, command := range commands {
		names = append(names, command.Name)
	}

	want := []string{"drinks", "ingredients", "inventory", "menus", "orders", "audit"}
	if !slices.Equal(names, want) {
		t.Fatalf("command nouns = %v, want %v", names, want)
	}
}

func TestTableColumns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		row  any
		want []string
	}{
		{"drink", drinkscli.DrinkRow{}, []string{"ID", "NAME", "CATEGORY", "GLASS", "INGREDIENTS"}},
		{"ingredient", ingredientscli.IngredientRow{}, []string{"ID", "NAME", "CATEGORY", "UNIT", "DESCRIPTION"}},
		{"inventory", inventorycli.InventoryRow{}, []string{"INGREDIENT_ID", "QUANTITY", "UNIT", "COST_PER_UNIT", "LAST_UPDATED"}},
		{"menu", menuscli.MenuRow{}, []string{"ID", "NAME", "STATUS", "ITEMS", "CREATED_AT", "PUBLISHED_AT"}},
		{"menu item", menuscli.MenuItemRow{}, []string{"DRINK_ID", "DISPLAY_NAME", "PRICE", "FEATURED", "AVAILABILITY", "SORT_ORDER"}},
		{"order", orderscli.OrderRow{}, []string{"ID", "MENU_ID", "STATUS", "ITEMS", "TOTAL_QUANTITY", "CREATED_AT", "COMPLETED_AT"}},
		{"order item", orderscli.OrderItemRow{}, []string{"DRINK_ID", "QUANTITY", "NOTES"}},
		{"audit", auditcli.AuditRow{}, []string{"ID", "STARTED_AT", "COMPLETED_AT", "DURATION", "ACTION", "RESOURCE", "PRINCIPAL", "SUCCESS", "TOUCHES", "ERROR"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			typ := reflect.TypeOf(tt.row)
			var got []string
			for field := range typ.Fields() {
				if column := field.Tag.Get("table"); column != "" && column != "-" {
					got = append(got, column)
				}
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("columns = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTableRowsIncludeDerivedValues(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, time.July, 22, 12, 0, 0, 0, time.UTC)
	completedAt := startedAt.Add(1500 * time.Millisecond)
	price := money.NewPriceFromCents(1234, currency.USD)

	drink := drinkscli.ToDrinkRow(&drinksmodels.Drink{
		Category: drinksmodels.DrinkCategoryCocktail,
		Glass:    drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{
			{}, {},
		}},
	})
	if drink.Category != "cocktail" || drink.Glass != "coupe" || drink.Ingredients != 2 {
		t.Fatalf("unexpected drink row: %+v", drink)
	}

	ingredient := ingredientscli.ToIngredientRow(&ingredientsmodels.Ingredient{Description: "Bright and tart"})
	if ingredient.Desc != "Bright and tart" {
		t.Fatalf("unexpected ingredient row: %+v", ingredient)
	}

	inventory := inventorycli.ToInventoryRow(&inventorymodels.Inventory{
		Amount:      measurement.MustAmount(3.5, measurement.UnitOz),
		CostPerUnit: optional.Some(price),
		LastUpdated: startedAt,
	})
	if inventory.CostPerUnit != "$12.34" || inventory.LastUpdated != "2026-07-22T12:00:00Z" {
		t.Fatalf("unexpected inventory row: %+v", inventory)
	}

	menuRows := menuscli.ToMenuItemRows([]menusmodels.MenuItem{{
		DrinkID:      entity.NewDrinkID(),
		DisplayName:  optional.Some("House Sour"),
		Price:        optional.Some(menusmodels.Price(price)),
		Featured:     true,
		Availability: menusmodels.AvailabilityLimited,
		SortOrder:    2,
	}})
	if len(menuRows) != 1 || menuRows[0].DisplayName != "House Sour" || menuRows[0].Price != "$12.34" || !menuRows[0].Featured || menuRows[0].SortOrder != 2 {
		t.Fatalf("unexpected menu item rows: %+v", menuRows)
	}

	order := orderscli.ToOrderRow(&ordersmodels.Order{
		Items:       []ordersmodels.OrderItem{{Quantity: 2}, {Quantity: 3}},
		CompletedAt: optional.Some(completedAt),
	})
	if order.Items != 2 || order.TotalQuantity != 5 || order.CompletedAt != "2026-07-22T12:00:01Z" {
		t.Fatalf("unexpected order row: %+v", order)
	}
	orderItems := orderscli.ToOrderItemRows([]ordersmodels.OrderItem{{Notes: "No garnish"}})
	if len(orderItems) != 1 || orderItems[0].Notes != "No garnish" {
		t.Fatalf("unexpected order item rows: %+v", orderItems)
	}

	audit := auditcli.ToAuditRow(&auditmodels.AuditEntry{StartedAt: startedAt, CompletedAt: completedAt})
	if audit.CompletedAt != "2026-07-22T12:00:01Z" || audit.Duration != "1.5s" {
		t.Fatalf("unexpected audit row: %+v", audit)
	}
}
