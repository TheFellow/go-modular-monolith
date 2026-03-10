package menus_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestMenu_CreateRejectsIDProvided(t *testing.T) {
	t.Parallel()
	fix := testutil.NewFixture(t)

	_, err := fix.Menu.Create(fix.OwnerContext(), &models.Menu{ID: models.NewMenuID("explicit-id")})
	testutil.ErrorIsInvalid(t, err)
}
