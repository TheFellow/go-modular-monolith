package dispatcher_test

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/inventory/events"
	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestDispatch_StockAdjusted_ReachesMenuHandler(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	prevOut := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	})

	ctx := middleware.NewContext(context.Background())
	d := dispatcher.New()

	err := d.Dispatch(ctx, events.StockAdjusted{
		IngredientID: cedar.NewEntityUID(cedar.EntityType("Mixology::Ingredient"), cedar.String("vodka")),
		PreviousQty:  10,
		NewQty:       0,
		Delta:        -10,
		Reason:       "used",
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	if !strings.Contains(buf.String(), "menu: ingredient depleted") {
		t.Fatalf("expected menu handler log, got: %q", buf.String())
	}
}
