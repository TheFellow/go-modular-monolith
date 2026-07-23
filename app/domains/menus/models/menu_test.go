package models_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestMenuValidateAcceptsWhitespacePaddedNameWithoutNormalizing(t *testing.T) {
	t.Parallel()

	menu := models.Menu{
		Name:   "  Dinner Service  ",
		Status: models.MenuStatusDraft,
	}

	testutil.Ok(t, menu.Validate())
	testutil.Equals(t, menu.Name, "  Dinner Service  ")
}
