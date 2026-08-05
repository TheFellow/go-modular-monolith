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

func TestMenuLifecyclePreconditions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		menu                   models.Menu
		check                  func(models.Menu) error
		wantFailedPrecondition bool
	}{
		{name: "draft mutation accepts draft", menu: models.Menu{Status: models.MenuStatusDraft}, check: models.Menu.RequireDraft},
		{name: "draft mutation rejects published", menu: models.Menu{Status: models.MenuStatusPublished}, check: models.Menu.RequireDraft, wantFailedPrecondition: true},
		{name: "publish accepts non-empty draft", menu: models.Menu{Status: models.MenuStatusDraft, Items: []models.MenuItem{{}}}, check: models.Menu.RequirePublishable},
		{name: "publish rejects empty draft", menu: models.Menu{Status: models.MenuStatusDraft}, check: models.Menu.RequirePublishable, wantFailedPrecondition: true},
		{name: "publish rejects published", menu: models.Menu{Status: models.MenuStatusPublished, Items: []models.MenuItem{{}}}, check: models.Menu.RequirePublishable, wantFailedPrecondition: true},
		{name: "return accepts published", menu: models.Menu{Status: models.MenuStatusPublished}, check: models.Menu.RequireReturnToDraft},
		{name: "return rejects draft", menu: models.Menu{Status: models.MenuStatusDraft}, check: models.Menu.RequireReturnToDraft, wantFailedPrecondition: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.check(tt.menu)
			if tt.wantFailedPrecondition {
				testutil.ErrorIsFailedPrecondition(t, err)
				return
			}
			testutil.Ok(t, err)
		})
	}
}
