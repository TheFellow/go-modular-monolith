package tui

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
)

type menuItem struct {
	menu        models.Menu
	description string
}

func newMenuItem(menu models.Menu, styles tui.ListViewStyles) menuItem {
	status := menuStatusBadge(menu.Status, styles)
	description := fmt.Sprintf("%s | %d drinks", status, len(menu.Items))
	return menuItem{menu: menu, description: description}
}

func (i menuItem) Title() string { return i.menu.Name }
func (i menuItem) Description() string {
	return i.description
}
func (i menuItem) FilterValue() string { return i.menu.Name }
