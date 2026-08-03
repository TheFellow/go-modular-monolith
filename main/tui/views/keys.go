package views

import (
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/charmbracelet/bubbles/key"
)

type dashboardKeys struct {
	Nav1, Nav2, Nav3, Nav4, Nav5, Nav6, Nav7 key.Binding
	Refresh, Help, Quit                      key.Binding
}

func newDashboardKeys() dashboardKeys {
	return dashboardKeys{
		Nav1:    keys.NewBinding("1", "drinks", "1"),
		Nav2:    keys.NewBinding("2", "ingredients", "2"),
		Nav3:    keys.NewBinding("3", "inventory", "3"),
		Nav4:    keys.NewBinding("4", "menus", "4"),
		Nav5:    keys.NewBinding("5", "orders", "5"),
		Nav6:    keys.NewBinding("6", "audit", "6"),
		Nav7:    keys.NewBinding("7", "tags", "7"),
		Refresh: keys.Standard.Refresh,
		Help:    keys.Standard.Help,
		Quit:    keys.Standard.Quit,
	}
}
