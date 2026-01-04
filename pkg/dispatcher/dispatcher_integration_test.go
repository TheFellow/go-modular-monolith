package dispatcher_test

import (
	"context"
	"testing"
	"time"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/mjl-/bstore"
)

func TestDispatch_StockAdjusted_UpdatesMenuAvailability(t *testing.T) {
	testutil.OpenStore(t)
	d := dispatcher.New()

	err := store.DB.Write(context.Background(), func(tx *bstore.Tx) error {
		ctx := middleware.NewContext(context.Background(), middleware.WithTransaction(tx))

		drink := drinksmodels.Drink{
			ID:   "margarita",
			Name: "Margarita",
			Recipe: drinksmodels.Recipe{
				Ingredients: []drinksmodels.RecipeIngredient{
					{IngredientID: ingredientsmodels.NewIngredientID("vodka"), Amount: 1, Unit: ingredientsmodels.UnitOz},
				},
			},
		}
		if err := tx.Insert(&drink); err != nil {
			return err
		}

		menu := menumodels.Menu{
			ID:        "happy-hour",
			Name:      "Happy Hour",
			Status:    menumodels.MenuStatusPublished,
			CreatedAt: time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC),
			Items: []menumodels.MenuItem{
				{DrinkID: drinksmodels.NewDrinkID("margarita"), Availability: menumodels.AvailabilityAvailable},
			},
		}
		if err := tx.Insert(&menu); err != nil {
			return err
		}

		return d.Dispatch(ctx, events.StockAdjusted{
			IngredientID: ingredientsmodels.NewIngredientID("vodka"),
			PreviousQty:  10,
			NewQty:       0,
			Delta:        -10,
			Reason:       "used",
		})
	})
	if err != nil {
		t.Fatalf("write tx: %v", err)
	}

	var got menumodels.Menu
	if err := store.DB.Read(context.Background(), func(tx *bstore.Tx) error {
		got = menumodels.Menu{ID: "happy-hour"}
		return tx.Get(&got)
	}); err != nil {
		t.Fatalf("read menu: %v", err)
	}

	if len(got.Items) != 1 {
		t.Fatalf("expected 1 menu item, got %d", len(got.Items))
	}
	if string(got.Items[0].DrinkID.ID) != "margarita" {
		t.Fatalf("expected menu item drink_id margarita, got %q", string(got.Items[0].DrinkID.ID))
	}
	if got.Items[0].Availability != menumodels.AvailabilityUnavailable {
		t.Fatalf("expected menu item availability unavailable, got %q", got.Items[0].Availability)
	}
}
