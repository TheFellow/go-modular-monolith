package commands

import "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"

func ensureDraftMenu(menu *models.Menu) error {
	return menu.RequireDraft()
}
