package tui

import (
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/charmbracelet/bubbles/key"
)

type listViewKeys struct {
	keys.ListViewKeys
	Tags, Adjust, Set, Quarantine, Release, Dispose, History key.Binding
}

func newListViewKeys() listViewKeys {
	return listViewKeys{
		ListViewKeys: keys.Standard.ListView,
		Quarantine:   keys.NewBinding("x", "quarantine", "x"),
		Release:      keys.NewBinding("u", "release quarantine", "u"),
		Dispose:      keys.NewBinding("d", "dispose", "d"),
		History:      keys.NewBinding("h", "stock history", "h"),
		Tags:         keys.NewBinding("t", "manage tags", "t"),
		Adjust:       keys.NewBinding("a", "adjust", "a"),
		Set:          keys.NewBinding("s", "set", "s"),
	}
}
