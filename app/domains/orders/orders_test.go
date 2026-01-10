package orders_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestOrders_PlaceRejectsIDProvided(t *testing.T) {
	fix := testutil.NewFixture(t)

	_, err := fix.Orders.Place(fix.Ctx, models.Order{ID: models.NewOrderID("explicit-id")})
	testutil.ErrorIf(t, err == nil || !errors.IsInvalid(err), "expected invalid error, got %v", err)
}
