package tui

import (
	tuikeys "github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/charmbracelet/bubbles/key"
)

type listViewKeys struct {
	tuikeys.ListViewKeys
	Tags, Adjust, Set key.Binding
}

func newListViewKeys() listViewKeys {
	return listViewKeys{
		ListViewKeys: tuikeys.Standard.ListView,
		Tags:         tuikeys.NewBinding("t", "manage tags", "t"),
		Adjust:       tuikeys.NewBinding("a", "adjust", "a"),
		Set:          tuikeys.NewBinding("s", "set", "s"),
	}
}
