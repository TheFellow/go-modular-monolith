package views

import (
	tuikeys "github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/charmbracelet/bubbles/key"
)

type dashboardKeys struct {
	Nav1, Nav2, Nav3, Nav4, Nav5, Nav6, Nav7 key.Binding
	Refresh, Help, Quit                      key.Binding
}

func newDashboardKeys() dashboardKeys {
	return dashboardKeys{
		Nav1:    tuikeys.NewBinding("1", "drinks", "1"),
		Nav2:    tuikeys.NewBinding("2", "ingredients", "2"),
		Nav3:    tuikeys.NewBinding("3", "inventory", "3"),
		Nav4:    tuikeys.NewBinding("4", "menus", "4"),
		Nav5:    tuikeys.NewBinding("5", "orders", "5"),
		Nav6:    tuikeys.NewBinding("6", "audit", "6"),
		Nav7:    tuikeys.NewBinding("7", "tags", "7"),
		Refresh: tuikeys.Standard.Refresh,
		Help:    tuikeys.Standard.Help,
		Quit:    tuikeys.Standard.Quit,
	}
}
