package tui

import "github.com/TheFellow/go-modular-monolith/pkg/tui"

// ListViewStylesFrom creates ListViewStyles from the main Styles.
func ListViewStylesFrom(s Styles) tui.ListViewStyles {
	return tui.ListViewStyles{
		Title:       s.Title,
		Subtitle:    s.Subtitle,
		Muted:       s.Unselected,
		Selected:    s.Selected,
		ListPane:    s.ListPane,
		DetailPane:  s.DetailPane,
		ErrorText:   s.ErrorText,
		WarningText: s.WarningText,
	}
}

// ListViewKeysFrom creates ListViewKeys from the main KeyMap.
func ListViewKeysFrom(k KeyMap) tui.ListViewKeys {
	return tui.ListViewKeys{
		Up:      k.Up,
		Down:    k.Down,
		Enter:   k.Enter,
		Refresh: k.Refresh,
		Back:    k.Back,
	}
}
