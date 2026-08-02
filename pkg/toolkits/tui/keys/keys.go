package keys

import (
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/dialog"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keyname"
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines all key bindings for the TUI.
type KeyMap struct {
	// Global bindings
	Quit key.Binding
	Help key.Binding
	Back key.Binding

	// List navigation (used by list views)
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Refresh key.Binding
	Create  key.Binding
	Edit    key.Binding
	Delete  key.Binding

	// Form keys
	NextField key.Binding
	PrevField key.Binding
	Submit    key.Binding

	// Dialog keys
	Confirm   key.Binding
	SwitchBtn key.Binding

	// Derived subsets
	ListView ListViewKeys
	Form     forms.FormKeys
	Dialog   dialog.DialogKeys
}

// Standard is the reusable default key map.
var Standard = newKeyMap()

// ListViewKeys defines standard list navigation and CRUD bindings.
type ListViewKeys struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Refresh key.Binding
	Back    key.Binding
	Create  key.Binding
	Edit    key.Binding
	Delete  key.Binding
}

// NewBinding creates a binding while keeping key/help construction consistent.
func NewBinding(helpKey, help string, values ...string) key.Binding {
	return key.NewBinding(key.WithKeys(values...), key.WithHelp(helpKey, help))
}

// newKeyMap creates a KeyMap with default bindings.
func newKeyMap() KeyMap {
	keys := KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Back: key.NewBinding(
			key.WithKeys(keyname.Escape),
			key.WithHelp(keyname.Escape, "back"),
		),
		Up: key.NewBinding(
			key.WithKeys(keyname.Up, keyname.VimUp),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys(keyname.Down, keyname.VimDown),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys(keyname.Enter),
			key.WithHelp(keyname.Enter, "select"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Create: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "create"),
		),
		Edit: key.NewBinding(
			key.WithKeys(keyname.Edit),
			key.WithHelp(keyname.Edit, "edit"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		NextField: key.NewBinding(
			key.WithKeys(keyname.Down, keyname.VimDown, keyname.Tab),
			key.WithHelp("↓/j", "next field"),
		),
		PrevField: key.NewBinding(
			key.WithKeys(keyname.Up, keyname.VimUp, keyname.ShiftTab),
			key.WithHelp("↑/k", "previous field"),
		),
		Submit: key.NewBinding(
			key.WithKeys(keyname.Submit),
			key.WithHelp(keyname.Submit, "submit"),
		),
		Confirm: key.NewBinding(
			key.WithKeys(keyname.Enter),
			key.WithHelp(keyname.Enter, "confirm"),
		),
		SwitchBtn: key.NewBinding(
			key.WithKeys(keyname.Tab, keyname.Left, keyname.Right),
			key.WithHelp("tab/←/→", "switch"),
		),
	}

	keys.ListView = listViewKeysFrom(keys)
	keys.Form = formKeysFrom(keys)
	keys.Dialog = dialogKeysFrom(keys)

	return keys
}

// ShortHelp returns bindings shown in the mini help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Refresh, k.Back, k.Quit}
}

// FullHelp returns bindings shown in the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Refresh},
		{k.Back, k.Help, k.Quit},
	}
}

func listViewKeysFrom(k KeyMap) ListViewKeys {
	return ListViewKeys{
		Up:      k.Up,
		Down:    k.Down,
		Enter:   k.Enter,
		Refresh: k.Refresh,
		Back:    k.Back,
		Create:  k.Create,
		Edit:    k.Edit,
		Delete:  k.Delete,
	}
}

func formKeysFrom(k KeyMap) forms.FormKeys {
	return forms.FormKeys{
		NextField: k.NextField,
		PrevField: k.PrevField,
		Edit:      k.Edit,
		Accept:    k.Enter,
		Submit:    k.Submit,
		Cancel:    k.Back,
	}
}

func dialogKeysFrom(k KeyMap) dialog.DialogKeys {
	return dialog.DialogKeys{
		Confirm: k.Confirm,
		Cancel:  k.Back,
		Switch:  k.SwitchBtn,
	}
}
