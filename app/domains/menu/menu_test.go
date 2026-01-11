package menu_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestMenu_CreateRejectsIDProvided(t *testing.T) {
	fix := testutil.NewFixture(t)

	_, err := fix.Menu.Create(fix.OwnerContext(), models.Menu{ID: models.NewMenuID("explicit-id")})
	testutil.ErrorIf(t, err == nil || !errors.IsInvalid(err), "expected invalid error, got %v", err)
}
