package forms_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func TestFormNavigation(t *testing.T) {
	keys := forms.FormKeys{
		NextField: key.NewBinding(key.WithKeys("tab")),
		PrevField: key.NewBinding(key.WithKeys("shift+tab")),
	}
	form := forms.New(forms.FormStyles{}, keys,
		forms.NewTextField("Name"),
		forms.NewTextField("Email"),
	)
	form.Init()
	if field := form.FocusedField(); field == nil || field.Label() != "Name" {
		t.Fatalf("expected first field focused")
	}

	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyTab})
	if field := form.FocusedField(); field == nil || field.Label() != "Email" {
		t.Fatalf("expected second field focused")
	}

	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if field := form.FocusedField(); field == nil || field.Label() != "Name" {
		t.Fatalf("expected first field focused after shift+tab")
	}
}

func TestFormValidation(t *testing.T) {
	keys := forms.FormKeys{}
	name := forms.NewTextField("Name", forms.WithRequired())
	age := forms.NewNumberField("Age", forms.WithMin(21))
	form := forms.New(forms.FormStyles{}, keys, name, age)

	if err := name.SetValue("Ada"); err != nil {
		t.Fatalf("set value: %v", err)
	}
	if err := age.SetValue("18"); err != nil {
		t.Fatalf("set value: %v", err)
	}

	if err := form.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
	if age.Error() == nil {
		t.Fatalf("expected age field error")
	}

	if err := age.SetValue("21"); err != nil {
		t.Fatalf("set value: %v", err)
	}
	if err := form.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}
