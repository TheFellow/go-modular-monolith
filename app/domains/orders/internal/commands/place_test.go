package commands_test

import (
	"context"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestPlace_RejectsIDProvided(t *testing.T) {
	cmds := commands.New()
	ctx := middleware.NewContext(context.Background())

	_, err := cmds.Place(ctx, models.Order{ID: models.NewOrderID("explicit-id")})
	testutil.ErrorIf(t, err == nil || !errors.IsInvalid(err), "expected invalid error, got %v", err)
}
