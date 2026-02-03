package tui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	tuikeys "github.com/TheFellow/go-modular-monolith/main/tui/keys"
	tuistyles "github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// AdjustInventoryVM renders an inventory adjustment form.
type AdjustInventoryVM struct {
	app        *app.App
	form       *forms.Form
	row        InventoryRow
	styles     forms.FormStyles
	keys       forms.FormKeys
	err        error
	submitting bool
	amount     *forms.NumberField
	reason     *forms.SelectField
}

// AdjustErrorMsg is sent when adjusting inventory fails.
type AdjustErrorMsg struct {
	Err error
}

// NewAdjustInventoryVM builds an AdjustInventoryVM with fields configured.
func NewAdjustInventoryVM(app *app.App, row InventoryRow) *AdjustInventoryVM {
	reasonOptions := []forms.SelectOption{
		{Label: "Received", Value: models.ReasonReceived},
		{Label: "Used", Value: models.ReasonUsed},
		{Label: "Spilled", Value: models.ReasonSpilled},
		{Label: "Expired", Value: models.ReasonExpired},
		{Label: "Corrected", Value: models.ReasonCorrected},
	}

	amountField := forms.NewNumberField(
		"Amount",
		forms.WithRequired(),
		forms.WithPrecision(2),
		forms.WithAllowNegative(),
		forms.WithPlaceholder("e.g., +5.0 or -2.5"),
	)
	reasonField := forms.NewSelectField(
		"Reason",
		reasonOptions,
		forms.WithRequired(),
	)

	formStyles := tuistyles.App.Form
	formKeys := tuikeys.App.Form
	form := forms.New(
		formStyles,
		formKeys,
		amountField,
		reasonField,
	)

	return &AdjustInventoryVM{
		app:    app,
		form:   form,
		row:    row,
		styles: formStyles,
		keys:   formKeys,
		amount: amountField,
		reason: reasonField,
	}
}

// Init initializes the form.
func (m *AdjustInventoryVM) Init() tea.Cmd {
	return m.form.Init()
}

// Update handles messages for the form.
func (m *AdjustInventoryVM) Update(msg tea.Msg) (*AdjustInventoryVM, tea.Cmd) {
	switch typed := msg.(type) {
	case AdjustErrorMsg:
		m.submitting = false
		m.err = typed.Err
		return m, nil
	case InventoryAdjustedMsg:
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
func (m *AdjustInventoryVM) View() string {
	title := "Adjust Inventory"
	if name := strings.TrimSpace(m.row.Ingredient.Name); name != "" {
		title = "Adjust: " + name
	}

	current := "Current: N/A"
	if m.row.Inventory.Amount != nil {
		current = fmt.Sprintf("Current: %.2f %s", m.row.Inventory.Amount.Value(), m.row.Inventory.Amount.Unit())
	}

	view := strings.Join([]string{title, current, "", m.form.View()}, "\n")
	if m.err != nil {
		errText := m.styles.Error.Render("Error: " + m.err.Error())
		return strings.Join([]string{errText, "", view}, "\n")
	}
	return view
}

// SetWidth sets the width of the form.
func (m *AdjustInventoryVM) SetWidth(w int) {
	m.form.SetWidth(w)
}

// IsDirty reports whether the form has been modified.
func (m *AdjustInventoryVM) IsDirty() bool {
	return m.form.IsDirty()
}

func (m *AdjustInventoryVM) submit() tea.Cmd {
	if m.submitting {
		return nil
	}
	if err := m.form.Validate(); err != nil {
		m.err = err
		return nil
	}

	amountValue, ok := toFloat(m.amount.Value())
	if !ok {
		m.err = errors.New("amount is required")
		return nil
	}
	unit := unitFromRow(m.row)
	if unit == "" {
		m.err = errors.New("unit is required")
		return nil
	}

	amount, err := measurement.NewAmount(amountValue, unit)
	if err != nil {
		m.err = err
		return nil
	}

	patch := &models.Patch{
		IngredientID: m.row.Ingredient.ID,
		Reason:       toAdjustmentReason(m.reason.Value()),
		Delta:        optional.Some(amount),
		CostPerUnit:  optional.None[money.Price](),
	}
	m.err = nil
	m.submitting = true

	return func() tea.Msg {
		adjusted, err := m.app.Inventory.Adjust(m.context(), patch)
		if err != nil {
			return AdjustErrorMsg{Err: err}
		}
		return InventoryAdjustedMsg{Inventory: adjusted}
	}
}

func (m *AdjustInventoryVM) context() *middleware.Context {
	return m.app.Context()
}

func toAdjustmentReason(value any) models.AdjustmentReason {
	switch typed := value.(type) {
	case models.AdjustmentReason:
		return typed
	case string:
		return models.AdjustmentReason(typed)
	default:
		return ""
	}
}

func toFloat(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func unitFromRow(row InventoryRow) measurement.Unit {
	if row.Ingredient.Unit != "" {
		return row.Ingredient.Unit
	}
	if row.Inventory.Amount != nil {
		return row.Inventory.Amount.Unit()
	}
	return ""
}
