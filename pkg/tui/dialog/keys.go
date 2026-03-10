package dialog

import "github.com/charmbracelet/bubbles/key"

// DialogKeys defines key bindings for dialog navigation.
type DialogKeys struct {
	Confirm key.Binding
	Cancel  key.Binding
	Switch  key.Binding
}
