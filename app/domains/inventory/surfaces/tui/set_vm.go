package tui

import (
	"errors"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/app/surfaces/tui/components"
	tuikeys "github.com/TheFellow/go-modular-monolith/app/surfaces/tui/keys"
	tuistyles "github.com/TheFellow/go-modular-monolith/app/surfaces/tui/styles"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// SetInventoryVM renders an inventory set form.
type SetInventoryVM struct {
	app        *app.Session
	form       *forms.Form
	row        InventoryRow
	styles     forms.FormStyles
	keys       forms.FormKeys
	err        error
	submitting bool
	quantity   *forms.NumberField
	cost       *forms.TextField
	tags       *forms.TextField
}

// SetErrorMsg is sent when setting inventory fails.
type SetErrorMsg struct {
	Err error
}

// NewSetInventoryVM builds a SetInventoryVM with fields configured.
func NewSetInventoryVM(app *app.Session, row InventoryRow) *SetInventoryVM {
	quantityField := forms.NewNumberField(
		"Quantity",
		forms.WithRequired(),
		forms.WithPrecision(2),
		forms.WithMin(0),
	)
	if row.Inventory.Amount != nil {
		_ = quantityField.SetValue(row.Inventory.Amount.Value())
	}

	costField := forms.NewTextField("Cost Per Unit", forms.WithPlaceholder("Optional, e.g. $1.23 or EUR 1.23"))
	tagsField := components.NewOptionalTagsField(row.Inventory.Tags)

	formStyles := tuistyles.App.Form
	formKeys := tuikeys.App.Form
	form := forms.New(
		formStyles,
		formKeys,
		quantityField,
		costField,
		tagsField,
	)

	return &SetInventoryVM{
		app:      app,
		form:     form,
		row:      row,
		styles:   formStyles,
		keys:     formKeys,
		quantity: quantityField,
		cost:     costField,
		tags:     tagsField,
	}
}

// Init initializes the form.
func (m *SetInventoryVM) Init() tea.Cmd {
	return m.form.Init()
}

// Update handles messages for the form.
func (m *SetInventoryVM) Update(msg tea.Msg) (*SetInventoryVM, tea.Cmd) {
	switch typed := msg.(type) {
	case SetErrorMsg:
		m.submitting = false
		m.err = typed.Err
		return m, nil
	case InventorySetMsg:
		m.submitting = false
		m.err = nil
		return m, nil
	case tea.KeyMsg:
		if key.Matches(typed, m.keys.Submit) {
			return m, m.submit()
		}
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

// View renders the form with context.
func (m *SetInventoryVM) View() string {
	title := "Set Inventory"
	if name := strings.TrimSpace(m.row.Ingredient.Name); name != "" {
		title = "Set Inventory: " + name
	}
	unit := "Unit: N/A"
	if unitValue := unitFromRow(m.row); unitValue != "" {
		unit = "Unit: " + string(unitValue)
	}

	view := strings.Join([]string{title, unit, "", m.form.View()}, "\n")
	if m.err != nil {
		errText := m.styles.Error.Render("Error: " + m.err.Error())
		return strings.Join([]string{errText, "", view}, "\n")
	}
	return view
}

// SetWidth sets the width of the form.
func (m *SetInventoryVM) SetWidth(w int) {
	m.form.SetWidth(w)
}

// IsDirty reports whether the form has been modified.
func (m *SetInventoryVM) IsDirty() bool {
	return m.form.IsDirty()
}

func (m *SetInventoryVM) submit() tea.Cmd {
	if m.submitting {
		return nil
	}
	if err := m.form.Validate(); err != nil {
		m.err = err
		return nil
	}

	quantityValue, ok := toFloat(m.quantity.Value())
	if !ok {
		m.err = errors.New("quantity is required")
		return nil
	}
	unit := unitFromRow(m.row)
	if unit == "" {
		m.err = errors.New("unit is required")
		return nil
	}
	amount, err := measurement.NewAmount(quantityValue, unit)
	if err != nil {
		m.err = err
		return nil
	}

	cost, err := m.parseCost()
	if err != nil {
		m.err = err
		return nil
	}
	desired, err := components.DesiredTags(m.tags)
	if err != nil {
		m.err = err
		return nil
	}

	update := &models.Update{
		IngredientID: m.row.Ingredient.ID,
		Amount:       amount,
		CostPerUnit:  cost,
	}
	m.err = nil
	m.submitting = true

	return func() tea.Msg {
		updated, err := app.RunTaggedMutation(m.app.App, m.context(), desired, func(ctx *middleware.Context) (*models.Inventory, error) {
			return m.app.Inventory.Set(ctx, update)
		})
		if err != nil {
			return SetErrorMsg{Err: err}
		}
		return InventorySetMsg{Inventory: updated}
	}
}

func (m *SetInventoryVM) context() *middleware.Context {
	return m.app.Context()
}

func (m *SetInventoryVM) parseCost() (money.Price, error) {
	value := strings.TrimSpace(m.cost.Value().(string))
	if value == "" {
		if price, ok := m.row.Inventory.CostPerUnit.Unwrap(); ok {
			return price, nil
		}
		return money.ParsePrice("USD 0.00")
	}
	return money.ParsePrice(value)
}
