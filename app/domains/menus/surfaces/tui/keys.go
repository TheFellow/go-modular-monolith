package tui

import (
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/charmbracelet/bubbles/key"
)

type listViewKeys struct {
	keys.ListViewKeys
	Tags, Publish, Draft key.Binding
}

func newListViewKeys() listViewKeys {
	return listViewKeys{
		ListViewKeys: keys.Standard.ListView,
		Tags:         keys.NewBinding("t", "manage tags", "t"),
		Publish:      keys.NewBinding("p", "publish", "p"),
		Draft:        keys.NewBinding("u", "draft", "u"),
	}
}
