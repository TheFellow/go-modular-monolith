package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// ListViewStyles contains styles needed by domain list ViewModels.
type ListViewStyles struct {
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Muted       lipgloss.Style
	Selected    lipgloss.Style
	ListPane    lipgloss.Style
	DetailPane  lipgloss.Style
	ErrorText   lipgloss.Style
	WarningText lipgloss.Style
}

// ListViewKeys contains key bindings needed by domain list ViewModels.
type ListViewKeys struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Refresh key.Binding
	Back    key.Binding
}

// ListViewStylesFrom creates ListViewStyles from the main Styles.
func ListViewStylesFrom(s Styles) ListViewStyles {
	return ListViewStyles{
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
func ListViewKeysFrom(k KeyMap) ListViewKeys {
	return ListViewKeys{
		Up:      k.Up,
		Down:    k.Down,
		Enter:   k.Enter,
		Refresh: k.Refresh,
		Back:    k.Back,
	}
}
