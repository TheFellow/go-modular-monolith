package tui

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
)

type menuItem = tui.ListItem[models.Menu]

func newMenuItem(menu models.Menu, styles tui.ListViewStyles) menuItem {
	status := menuStatusBadge(menu.Status, styles)
	description := fmt.Sprintf("%s | %d drinks", status, len(menu.Items))
	return tui.NewListItem(menu, menu.Name, description, menu.Name)
}
