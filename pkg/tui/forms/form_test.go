package forms_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func TestFormNavigation(t *testing.T) {
	t.Parallel()

	keys := forms.FormKeys{
		NextField: key.NewBinding(key.WithKeys("tab")),
		PrevField: key.NewBinding(key.WithKeys("shift+tab")),
	}
	form := forms.New(forms.FormStyles{}, keys,
		forms.NewTextField("Name"),
		forms.NewTextField("Email"),
	)
	form.Init()
	field := form.FocusedField()
	testutil.NotNil(t, field)
	testutil.Equals(t, field.Label(), "Name")

	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyTab})
	field = form.FocusedField()
	testutil.NotNil(t, field)
	testutil.Equals(t, field.Label(), "Email")

	form, _ = form.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	field = form.FocusedField()
	testutil.NotNil(t, field)
	testutil.Equals(t, field.Label(), "Name")
}

func TestFormValidation(t *testing.T) {
	t.Parallel()

	keys := forms.FormKeys{}
	name := forms.NewTextField("Name", forms.WithRequired())
	age := forms.NewNumberField("Age", forms.WithMin(21))
	form := forms.New(forms.FormStyles{}, keys, name, age)

	testutil.Ok(t, name.SetValue("Ada"))
	testutil.Ok(t, age.SetValue("18"))

	testutil.NotNil(t, form.Validate())
	testutil.NotNil(t, age.Error())

	testutil.Ok(t, age.SetValue("21"))
	testutil.Ok(t, form.Validate())
}
