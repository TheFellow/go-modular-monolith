package commands

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func ensureDraftMenu(menu *models.Menu) error {
	if menu.Status == models.MenuStatusDraft {
		return nil
	}

	return errors.FailedPreconditionf("menu %q must be draft, got %q", menu.ID.String(), menu.Status)
}
