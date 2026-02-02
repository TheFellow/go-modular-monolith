package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all key bindings for the TUI.
type KeyMap struct {
	// Global bindings
	Quit key.Binding
	Help key.Binding
	Back key.Binding

	// Navigation (dashboard only)
	Nav1 key.Binding // Drinks
	Nav2 key.Binding // Ingredients
	Nav3 key.Binding // Inventory
	Nav4 key.Binding // Menus
	Nav5 key.Binding // Orders
	Nav6 key.Binding // Audit

	// List navigation (used by list views)
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Refresh key.Binding
	Create  key.Binding
	Edit    key.Binding
	Delete  key.Binding
	Adjust  key.Binding
	Set     key.Binding

	// Form keys
	NextField key.Binding
	PrevField key.Binding
	Submit    key.Binding

	// Dialog keys
	Confirm   key.Binding
	SwitchBtn key.Binding
}

// NewKeyMap creates a KeyMap with default bindings.
func NewKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Nav1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "drinks"),
		),
		Nav2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "ingredients"),
		),
		Nav3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "inventory"),
		),
		Nav4: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "menus"),
		),
		Nav5: key.NewBinding(
			key.WithKeys("5"),
			key.WithHelp("5", "orders"),
		),
		Nav6: key.NewBinding(
			key.WithKeys("6"),
			key.WithHelp("6", "audit"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
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
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Adjust: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "adjust"),
		),
		Set: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "set"),
		),
		NextField: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next"),
		),
		PrevField: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "previous"),
		),
		Submit: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "submit"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		SwitchBtn: key.NewBinding(
			key.WithKeys("tab", "left", "right"),
			key.WithHelp("tab/←/→", "switch"),
		),
	}
}

// ShortHelp returns bindings shown in the mini help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Refresh, k.Back, k.Quit}
}

// FullHelp returns bindings shown in the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Nav1, k.Nav2, k.Nav3, k.Nav4, k.Nav5, k.Nav6},
		{k.Up, k.Down, k.Enter, k.Refresh},
		{k.Back, k.Help, k.Quit},
	}
}
