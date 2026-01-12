package dispatcher

import (
	"context"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func TestDispatcher_DispatchesToHandlers(t *testing.T) {
	t.Parallel()

	d := New()
	ctx := middleware.NewContext(context.Background())

	event := events.IngredientCreated{
		Ingredient: models.Ingredient{
			ID:   entity.IngredientID("vodka"),
			Name: "Vodka",
		},
	}
	err := d.Dispatch(ctx, event)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
}

func TestDispatcher_IgnoresUnknownEvents(t *testing.T) {
	t.Parallel()

	type unknownEvent struct{}

	d := New()
	ctx := middleware.NewContext(context.Background())
	if err := d.Dispatch(ctx, unknownEvent{}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
}
