package tui

import (
	tuikeys "github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/charmbracelet/bubbles/key"
)

type listViewKeys struct {
	tuikeys.ListViewKeys
	Tags, Publish, Draft key.Binding
}

func newListViewKeys() listViewKeys {
	return listViewKeys{
		ListViewKeys: tuikeys.Standard.ListView,
		Tags:         tuikeys.NewBinding("t", "manage tags", "t"),
		Publish:      tuikeys.NewBinding("p", "publish", "p"),
		Draft:        tuikeys.NewBinding("u", "draft", "u"),
	}
}
