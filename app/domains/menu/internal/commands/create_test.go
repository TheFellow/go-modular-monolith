package commands_test

import (
	"context"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestCreate_RejectsIDProvided(t *testing.T) {
	cmds := commands.New()
	ctx := middleware.NewContext(context.Background())

	_, err := cmds.Create(ctx, models.Menu{ID: models.NewMenuID("explicit-id")})
	testutil.ErrorIf(t, err == nil || !errors.IsInvalid(err), "expected invalid error, got %v", err)
}
