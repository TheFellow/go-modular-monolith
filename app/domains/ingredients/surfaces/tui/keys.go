package tui

import (
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/charmbracelet/bubbles/key"
)

type listViewKeys struct {
	keys.ListViewKeys
	Tags key.Binding
}

func newListViewKeys() listViewKeys {
	return listViewKeys{ListViewKeys: keys.Standard.ListView, Tags: keys.NewBinding("t", "manage tags", "t")}
}
