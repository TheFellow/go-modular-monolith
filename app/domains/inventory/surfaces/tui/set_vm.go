package tui

import (
	"errors"
	"strconv"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// SetDeps defines dependencies for the set inventory form.
type SetDeps struct {
	FormStyles forms.FormStyles
	FormKeys   forms.FormKeys
	Ctx        *middleware.Context
	SetFunc    func(ctx *middleware.Context, update *models.Update) (*models.Inventory, error)
}

// SetInventoryVM renders an inventory set form.
type SetInventoryVM struct {
	form       *forms.Form
	row        InventoryRow
	deps       SetDeps
	err        error
	submitting bool
	quantity   *forms.NumberField
	cost       *forms.NumberField
}

// SetErrorMsg is sent when setting inventory fails.
type SetErrorMsg struct {
	Err error
}

// NewSetInventoryVM builds a SetInventoryVM with fields configured.
func NewSetInventoryVM(row InventoryRow, deps SetDeps) *SetInventoryVM {
	quantityField := forms.NewNumberField(
		"Quantity",
		forms.WithRequired(),
		forms.WithPrecision(2),
		forms.WithMin(0),
	)
	if row.Inventory.Amount != nil {
		_ = quantityField.SetValue(row.Inventory.Amount.Value())
	}

	costOpts := []forms.FieldOption{
		forms.WithPrecision(2),
		forms.WithMin(0),
		forms.WithPlaceholder("Optional"),
	}
	if price, ok := row.Inventory.CostPerUnit.Unwrap(); ok {
		if cents, err := price.Cents(); err == nil {
			costOpts = append(costOpts, forms.WithInitialValue(float64(cents)/100))
		}
	}
	costField := forms.NewNumberField("Cost Per Unit", costOpts...)

	form := forms.New(
		deps.FormStyles,
		deps.FormKeys,
		quantityField,
		costField,
	)

	return &SetInventoryVM{
		form:     form,
		row:      row,
		deps:     deps,
		quantity: quantityField,
		cost:     costField,
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
		if key.Matches(typed, m.deps.FormKeys.Submit) {
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
		errText := m.deps.FormStyles.Error.Render("Error: " + m.err.Error())
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
	if m.deps.SetFunc == nil {
		m.err = errors.New("set function not configured")
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

	update := &models.Update{
		IngredientID: m.row.Ingredient.ID,
		Amount:       amount,
		CostPerUnit:  cost,
	}
	m.err = nil
	m.submitting = true

	return func() tea.Msg {
		updated, err := m.deps.SetFunc(m.deps.Ctx, update)
		if err != nil {
			return SetErrorMsg{Err: err}
		}
		return InventorySetMsg{Inventory: updated}
	}
}

func (m *SetInventoryVM) parseCost() (money.Price, error) {
	value, ok := toFloat(m.cost.Value())
	if !ok {
		if price, ok := m.row.Inventory.CostPerUnit.Unwrap(); ok {
			return price, nil
		}
		return money.NewPriceFromCents(0, currency.USD), nil
	}

	curr := currency.USD
	if price, ok := m.row.Inventory.CostPerUnit.Unwrap(); ok {
		curr = price.Currency
	}
	amount := strconv.FormatFloat(value, 'f', 2, 64)
	return money.NewPrice(amount, curr)
}
