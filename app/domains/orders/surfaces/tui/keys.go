package tui

import (
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/charmbracelet/bubbles/key"
)

type listViewKeys struct {
	keys.ListViewKeys
	Tags, Complete, Cancel, Amend, Batch key.Binding
}

func newListViewKeys() listViewKeys {
	return listViewKeys{
		ListViewKeys: keys.Standard.ListView,
		Amend:        keys.NewBinding("a", "amend", "a"),
		Batch:        keys.NewBinding("b", "review amendments", "b"),
		Tags:         keys.NewBinding("t", "manage tags", "t"),
		Complete:     keys.NewBinding("o", "complete", "o"),
		Cancel:       keys.NewBinding("x", "cancel order", "x"),
	}
}
