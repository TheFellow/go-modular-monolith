package dispatcher_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestDispatcher_DispatchesToHandlers(t *testing.T) {
	t.Parallel()

	d := dispatcher.New(nil)
	base := authn.ToContext(log.ToContext(context.Background(), slog.Default()), authn.Anonymous())
	ctx := middleware.NewContext(base)

	event := events.IngredientCreated{
		Ingredient: models.Ingredient{
			ID:   entity.IngredientID(cedar.NewEntityUID(entity.TypeIngredient, cedar.String("vodka"))),
			Name: "Vodka",
		},
	}
	err := d.Dispatch(ctx, event)
	testutil.Ok(t, err)
}

func TestDispatcher_IgnoresUnknownEvents(t *testing.T) {
	t.Parallel()

	type unknownEvent struct{}

	d := dispatcher.New(nil)
	base := authn.ToContext(log.ToContext(context.Background(), slog.Default()), authn.Anonymous())
	ctx := middleware.NewContext(base)
	testutil.Ok(t, d.Dispatch(ctx, unknownEvent{}))
}
