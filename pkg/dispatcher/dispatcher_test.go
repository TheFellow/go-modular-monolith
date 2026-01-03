package dispatcher

import (
	"context"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/handlers"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
)

func TestDispatcher_DispatchesToHandlers(t *testing.T) {
	t.Parallel()

	handlers.IngredientCreatedCount.Store(0)
	handlers.IngredientCreatedAuditCount.Store(0)

	d := New()

	err := d.Dispatch(context.Background(), events.IngredientCreated{
		IngredientID: models.NewIngredientID("vodka"),
		Name:         "Vodka",
	})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	if got := handlers.IngredientCreatedCount.Load(); got != 1 {
		t.Fatalf("expected IngredientCreatedCounter to run once, got %d", got)
	}
	if got := handlers.IngredientCreatedAuditCount.Load(); got != 1 {
		t.Fatalf("expected IngredientCreatedAudit to run once, got %d", got)
	}
}

func TestDispatcher_IgnoresUnknownEvents(t *testing.T) {
	t.Parallel()

	handlers.IngredientCreatedCount.Store(0)
	handlers.IngredientCreatedAuditCount.Store(0)

	d := New()
	if err := d.Dispatch(context.Background(), struct{}{}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	if got := handlers.IngredientCreatedCount.Load(); got != 0 {
		t.Fatalf("expected IngredientCreatedCounter not to run, got %d", got)
	}
	if got := handlers.IngredientCreatedAuditCount.Load(); got != 0 {
		t.Fatalf("expected IngredientCreatedAudit not to run, got %d", got)
	}
}
