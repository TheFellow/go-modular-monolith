package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	toolkit "github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type lifecycleVM struct {
	viewport   toolkit.FormViewport
	app        *app.Session
	row        InventoryRow
	action     actions.ID
	form       *forms.Form
	reason     *forms.TextField
	amount     *forms.TextField
	err        error
	submitting bool
}

type lifecycleErrorMsg struct{ Err error }
type historyLoadedMsg struct {
	Movements []models.Movement
	Err       error
	Token     uint64
}

func newLifecycleVM(session *app.Session, row InventoryRow, action actions.ID) *lifecycleVM {
	reason := forms.NewTextField("Reason", forms.WithRequired())
	amount := forms.NewTextField("Quantity ("+string(row.Inventory.Amount.Unit())+")", forms.WithRequired())
	fields := []forms.Field{reason}
	if action == inventory.ControlDispose {
		fields = []forms.Field{amount, reason}
	}
	return &lifecycleVM{viewport: toolkit.NewFormViewport(), app: session, row: row, action: action, reason: reason, amount: amount, form: forms.New(styles.Standard.Form, keys.Standard.Form, fields...)}
}

func (m *lifecycleVM) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case lifecycleErrorMsg:
		m.err, m.submitting = msg.Err, false
		return nil
	case tea.KeyMsg:
		if key.Matches(msg, keys.Standard.Form.Submit) {
			return m.submit()
		}
	}
	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return cmd
}

func (m *lifecycleVM) submit() tea.Cmd {
	if m.submitting {
		return nil
	}
	if err := m.form.Validate(); err != nil {
		m.err = err
		return nil
	}
	reason := strings.TrimSpace(fmt.Sprint(m.reason.Value()))
	var amount measurement.Amount
	if m.action == inventory.ControlDispose {
		value, ok := toFloat(m.amount.Value())
		if !ok || value <= 0 || math.IsInf(value, 0) || math.IsNaN(value) {
			m.err = errors.Invalidf("positive disposal amount is required")
			return nil
		}
		var err error
		amount, err = measurement.NewAmount(value, m.row.Inventory.Amount.Unit())
		if err != nil {
			m.err = err
			return nil
		}
	}
	m.submitting, m.err = true, nil
	return func() tea.Msg {
		var stock *models.Inventory
		var err error
		if m.action == inventory.ControlDispose {
			stock, err = m.app.Inventory.Dispose(m.app.Context(), models.Disposal{IngredientID: m.row.Inventory.IngredientID, Revision: m.row.Inventory.Revision, Amount: amount, Reason: reason})
		} else {
			stock, err = m.app.Inventory.Disposition(m.app.Context(), models.Disposition{IngredientID: m.row.Inventory.IngredientID, Revision: m.row.Inventory.Revision, Quarantine: m.action == inventory.ControlQuarantine, Reason: reason})
		}
		if err != nil {
			return lifecycleErrorMsg{Err: err}
		}
		return InventoryAdjustedMsg{Inventory: stock}
	}
}

func (m *lifecycleVM) View() string {
	title := map[actions.ID]string{inventory.ControlQuarantine: "Quarantine stock", inventory.ControlRelease: "Release quarantine", inventory.ControlDispose: "Dispose stock"}[m.action]
	lines := []string{title + ": " + m.row.Ingredient.Name, "On hand: " + exactInventoryAmount(m.row.Inventory.Amount), "Reserved: " + exactInventoryAmount(m.row.Inventory.ReservedAmount()), "", m.form.View()}
	if m.err != nil {
		lines = append(lines, styles.Standard.Form.Error.Render("Error: "+m.err.Error()))
	}
	focusLine := 4
	if m.action == inventory.ControlDispose && m.reason.IsFocused() {
		focusLine += strings.Count(m.amount.View(), "\n") + 2
	}
	if m.err != nil {
		focusLine = strings.Count(strings.Join(lines, "\n"), "\n")
	}
	return m.viewport.View(strings.Join(lines, "\n"), focusLine, "")
}

func inventoryHistoryView(movements []models.Movement, err error) string {
	lines := []string{"Stock movement history", ""}
	if err != nil {
		return strings.Join(append(lines, "Error: "+err.Error()), "\n")
	}
	if len(movements) == 0 {
		return strings.Join(append(lines, "No stock movements recorded"), "\n")
	}
	for _, movement := range movements {
		lines = append(lines, formatInventoryTime(movement.At), fmt.Sprintf("Before: %g %s; After: %g %s", movement.Before, movement.Unit, movement.After, movement.Unit), "Before: "+string(movement.BeforeStatus)+"; After: "+string(movement.AfterStatus), "Reason: "+movement.Reason, "Movement ID: "+movement.ID, "")
	}
	return strings.Join(lines, "\n")
}
