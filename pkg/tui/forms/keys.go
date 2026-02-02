package forms

import "github.com/charmbracelet/bubbles/key"

// FormKeys defines key bindings for form navigation.
type FormKeys struct {
	NextField key.Binding
	PrevField key.Binding
	Submit    key.Binding
	Cancel    key.Binding
}
