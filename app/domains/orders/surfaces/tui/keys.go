package tui

import (
	tuikeys "github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/charmbracelet/bubbles/key"
)

type listViewKeys struct {
	tuikeys.ListViewKeys
	Tags, Complete, Cancel key.Binding
}

func newListViewKeys() listViewKeys {
	return listViewKeys{
		ListViewKeys: tuikeys.Standard.ListView,
		Tags:         tuikeys.NewBinding("t", "manage tags", "t"),
		Complete:     tuikeys.NewBinding("o", "complete", "o"),
		Cancel:       tuikeys.NewBinding("x", "cancel order", "x"),
	}
}
