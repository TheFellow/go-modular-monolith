package tui

import (
	"cmp"
	"fmt"
	"strings"
	"time"

	"github.com/TheFellow/go-modular-monolith/app"
	menusmodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	orders "github.com/TheFellow/go-modular-monolith/app/domains/orders"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
	"github.com/charmbracelet/lipgloss"
	"github.com/govalues/decimal"
)

// DetailViewModel renders an order detail pane.
type DetailViewModel struct {
	styles  tui.ListViewStyles
	width   int
	height  int
	order   optional.Value[models.Order]
	app     *app.Session
	actions map[actions.ID]actions.State
}

func NewDetailViewModel(styles tui.ListViewStyles, app *app.Session) *DetailViewModel {
	return &DetailViewModel{
		styles: styles,
		app:    app,
	}
}

func (d *DetailViewModel) SetSize(width, height int) {
	d.width = width
	d.height = height
}

func (d *DetailViewModel) SetOrder(order optional.Value[models.Order]) {
	d.order = order
}

func (d *DetailViewModel) SetActions(states map[actions.ID]actions.State) { d.actions = states }

func (d *DetailViewModel) View() string {
	order, ok := d.order.Unwrap()
	if !ok {
		return d.styles.Subtitle.Render("Select an order to view details")
	}

	menu := &menusmodels.Menu{Name: order.Acceptance.MenuName}

	statusBadge := orderStatusBadge(order.Status, d.styles)
	lines := []string{
		d.styles.Title.Render("Order"),
		d.styles.Muted.Render("ID: " + order.ID.String()),
		d.styles.Subtitle.Render("Menu: ") + menu.Name,
		d.styles.Subtitle.Render("Status: ") + statusBadge,
		d.styles.Subtitle.Render("Tags: ") + cmp.Or(order.Tags.Canonical().String(), "(none)"),
		d.styles.Muted.Render("Created: " + formatTime(order.CreatedAt)),
	}

	if completedAt, ok := order.CompletedAt.Unwrap(); ok {
		lines = append(lines, d.styles.Muted.Render("Completed: "+formatTime(completedAt)))
	}
	if len(order.BlockedIngredients) > 0 {
		ids := make([]string, 0, len(order.BlockedIngredients))
		for _, id := range order.BlockedIngredients {
			ids = append(ids, id.String())
		}
		lines = append(lines, d.styles.ErrorText.Render("Short of reserved stock: "+strings.Join(ids, ", ")))
	}

	if strings.TrimSpace(order.Notes) != "" {
		lines = append(lines, "", d.styles.Subtitle.Render("Notes"), order.Notes)
	}
	for _, action := range []struct {
		id    actions.ID
		label string
	}{
		{orders.ControlComplete, "Complete"}, {orders.ControlCancel, "Cancel order"},
	} {
		if state, ok := d.actions[action.id]; ok && state.Visible && !state.Enabled && state.DisabledReason != "" {
			lines = append(lines, d.styles.Muted.Render(action.label+": "+state.DisabledReason))
		}
	}

	itemLines, total, err := d.renderSnapshot(order.Acceptance.Items)
	if err != nil {
		lines = append(lines, d.styles.ErrorText.Render(fmt.Sprintf("Error: %v", err)))
	} else {
		lines = append(lines, "", d.styles.Subtitle.Render("Items"))
		lines = append(lines, itemLines...)
		lines = append(lines, "", d.styles.Subtitle.Render("Total: ")+total)
	}

	lines = append(lines, "", "Approved preparation")
	for _, item := range order.Plan {
		lines = append(lines, item.Name)
		for _, selection := range item.Ingredients {
			if selection.Omitted {
				lines = append(lines, "  Omitted optional ingredient: "+selection.OriginalID.String())
				continue
			}
			lines = append(lines, fmt.Sprintf("  %g %s %s", selection.Quantity, selection.Unit, selection.Name))
		}
		lines = append(lines, item.Steps...)
		if item.Garnish != "" {
			lines = append(lines, "Garnish: "+item.Garnish)
		}
	}
	for _, amendment := range order.Amendments {
		lines = append(lines, "Amended "+formatTime(amendment.At)+" by "+amendment.Principal+": "+amendment.Reason)
	}
	if at, ok := order.CancelledAt.Unwrap(); ok {
		lines = append(lines, "Cancelled: "+formatTime(at), order.CancellationReason)
	}
	content := strings.Join(lines, "\n")
	if d.width > 0 {
		content = lipgloss.NewStyle().Width(d.width).Render(content)
	}
	return content
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func (d *DetailViewModel) renderSnapshot(items []models.ItemSnapshot) ([]string, string, error) {
	lines := []string{}
	var total menusmodels.Price
	known := true
	set := false
	for _, item := range items {
		lineTotal := "N/A"
		if price, ok := item.Price.Unwrap(); ok {
			qty, err := decimal.New(int64(item.Quantity), 0)
			if err != nil {
				return nil, "", err
			}
			cost, err := price.Mul(qty)
			if err != nil {
				return nil, "", err
			}
			lineTotal = cost.String()
			if !set {
				total = cost
				set = true
			} else {
				total, err = total.Add(cost)
				if err != nil {
					return nil, "", err
				}
			}
		} else {
			known = false
		}
		line := fmt.Sprintf("- %s | qty: %d | total: %s", item.Name, item.Quantity, lineTotal)
		if notes := strings.TrimSpace(item.Notes); notes != "" {
			line += "\n  Notes: " + strings.ReplaceAll(notes, "\n", "\n  ")
		}
		lines = append(lines, line)
	}
	if !known || !set {
		return lines, "N/A", nil
	}
	return lines, total.String(), nil
}
