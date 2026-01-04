package dispatcher_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/uow"
)

func TestDispatch_StockAdjusted_UpdatesMenuAvailability(t *testing.T) {
	tmp := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	dataDir := filepath.Join(tmp, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dataDir, "drinks.json"), []byte(`[
  {
    "id": "margarita",
    "name": "Margarita",
    "recipe": {
      "ingredients": [
        { "ingredient_id": "vodka", "amount": 1, "unit": "oz" }
      ]
    }
  }
]
`), 0o644); err != nil {
		t.Fatalf("write drinks fixture: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dataDir, "menus.json"), []byte(`[
  {
    "id": "happy-hour",
    "name": "Happy Hour",
    "status": "published",
    "created_at": "2026-01-04T00:00:00Z",
    "items": [
      { "drink_id": "margarita", "availability": "available" }
    ]
  }
]
`), 0o644); err != nil {
		t.Fatalf("write menus fixture: %v", err)
	}

	ctx := middleware.NewContext(context.Background())
	tx, err := uow.NewManager().Begin(ctx)
	if err != nil {
		t.Fatalf("begin uow: %v", err)
	}
	middleware.WithUnitOfWork(tx)(ctx)

	d := dispatcher.New()
	if err := d.Dispatch(ctx, events.StockAdjusted{
		IngredientID: ingredientsmodels.NewIngredientID("vodka"),
		PreviousQty:  10,
		NewQty:       0,
		Delta:        -10,
		Reason:       "used",
	}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(dataDir, "menus.json"))
	if err != nil {
		t.Fatalf("read saved menus: %v", err)
	}

	var menus []struct {
		Items []struct {
			DrinkID      string `json:"drink_id"`
			Availability string `json:"availability"`
		} `json:"items"`
	} // minimal shape for assertion
	if err := json.Unmarshal(b, &menus); err != nil {
		t.Fatalf("parse saved menus: %v", err)
	}

	if len(menus) != 1 {
		t.Fatalf("expected 1 menu, got %d", len(menus))
	}
	if len(menus[0].Items) != 1 {
		t.Fatalf("expected 1 menu item, got %d", len(menus[0].Items))
	}
	if menus[0].Items[0].DrinkID != "margarita" {
		t.Fatalf("expected menu item drink_id margarita, got %q", menus[0].Items[0].DrinkID)
	}
	if menus[0].Items[0].Availability != "unavailable" {
		t.Fatalf("expected menu item availability unavailable, got %q", menus[0].Items[0].Availability)
	}
}
