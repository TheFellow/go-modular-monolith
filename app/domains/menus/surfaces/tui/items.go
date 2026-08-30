package tui

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
)

type menuItem = tui.ListItem[models.Menu]

func newMenuItem(menu models.Menu, styles tui.ListViewStyles) menuItem {
	status := menuStatusBadge(menu.Status, styles)
	description := tui.ListSummary(status, fmt.Sprintf("%d drinks", len(menu.Items)), tui.TagSummary(menu.Tags.Canonical().String()))
	return tui.NewListItem(menu, menu.Name, description, menu.Name)
}
