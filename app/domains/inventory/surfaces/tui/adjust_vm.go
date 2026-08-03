package tui

import (
	"errors"
	"fmt"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"strconv"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/components"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// AdjustInventoryVM renders an inventory adjustment form.
type AdjustInventoryVM struct {
	app        *app.Session
	form       *forms.Form
	row        InventoryRow
	styles     forms.FormStyles
	keys       forms.FormKeys
	err        error
	submitting bool
	amount     *forms.NumberField
	cost       *forms.TextField
	reason     *forms.SelectField
	tags       *forms.TextField
}

// AdjustErrorMsg is sent when adjusting inventory fails.
type AdjustErrorMsg struct {
	Err error
}

// NewAdjustInventoryVM builds an AdjustInventoryVM with fields configured.
func NewAdjustInventoryVM(app *app.Session, row InventoryRow) *AdjustInventoryVM {
	reasonOptions := []forms.SelectOption{
		{Label: "Received", Value: models.ReasonReceived},
		{Label: "Used", Value: models.ReasonUsed},
		{Label: "Spilled", Value: models.ReasonSpilled},
		{Label: "Expired", Value: models.ReasonExpired},
		{Label: "Corrected", Value: models.ReasonCorrected},
	}

	amountField := forms.NewNumberField(
		"Delta",
		forms.WithPrecision(2),
		forms.WithAllowNegative(),
		forms.WithPlaceholder("e.g., +5.0 or -2.5"),
	)
	costField := forms.NewTextField("Cost per unit", forms.WithPlaceholder("e.g., $1.23 or EUR 1.23"))
	reasonField := forms.NewSelectField(
		"Reason",
		reasonOptions,
		forms.WithRequired(),
	)
	tagsField := components.NewOptionalTagsField(row.Inventory.Tags.Canonical().String())

	formStyles := styles.Standard.Form
	formKeys := keys.Standard.Form
	form := forms.New(
		formStyles,
		formKeys,
		amountField,
		costField,
		reasonField,
		tagsField,
	)

	return &AdjustInventoryVM{
		app:    app,
		form:   form,
		row:    row,
		styles: formStyles,
		keys:   formKeys,
		amount: amountField,
		cost:   costField,
		reason: reasonField,
		tags:   tagsField,
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
	if price, ok := m.row.Inventory.CostPerUnit.Unwrap(); ok {
		current += " at " + price.String()
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

	unit := unitFromRow(m.row)
	var delta optional.Value[measurement.Amount]
	if rawValue := m.amount.Value(); rawValue != nil {
		raw := strings.TrimSpace(fmt.Sprint(rawValue))
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			m.err = errors.New("delta must be a number")
			return nil
		}
		amount, err := measurement.NewAmount(value, unit)
		if err != nil {
			m.err = err
			return nil
		}
		delta = optional.Some(amount)
	}
	var cost optional.Value[money.Price]
	if raw := strings.TrimSpace(fmt.Sprint(m.cost.Value())); raw != "" {
		price, err := money.ParsePrice(raw)
		if err != nil {
			m.err = err
			return nil
		}
		cost = optional.Some(price)
	}
	if delta.IsNone() && cost.IsNone() {
		m.err = errors.New("at least one of delta or cost per unit is required")
		return nil
	}
	desired, err := components.DesiredTags(m.tags, tag.ParseCollection)
	if err != nil {
		m.err = err
		return nil
	}

	patch := &models.Patch{
		IngredientID: m.row.Ingredient.ID,
		Reason:       toAdjustmentReason(m.reason.Value()),
		Delta:        delta,
		CostPerUnit:  cost,
	}
	m.err = nil
	m.submitting = true

	return func() tea.Msg {
		adjusted, err := app.RunTaggedMutation(m.app.App, m.context(), desired, func(ctx *middleware.Context) (*models.Inventory, error) {
			return m.app.Inventory.Adjust(ctx, patch)
		})
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
	parsed, err := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(value)), 64)
	return parsed, err == nil
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
