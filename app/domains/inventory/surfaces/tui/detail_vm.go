package tui

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/main/tui/components"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/charmbracelet/lipgloss"
)

// DetailViewModel renders an inventory detail pane.
type DetailViewModel struct {
	styles tui.ListViewStyles
	width  int
	height int
	row    optional.Value[InventoryRow]
}

func NewDetailViewModel(styles tui.ListViewStyles) *DetailViewModel {
	return &DetailViewModel{styles: styles}
}

func (d *DetailViewModel) SetSize(width, height int) {
	d.width = width
	d.height = height
}

func (d *DetailViewModel) SetRow(row optional.Value[InventoryRow]) {
	d.row = row
}

func (d *DetailViewModel) View() string {
	row, ok := d.row.Unwrap()
	if !ok {
		return d.styles.Subtitle.Render("Select a stock item to view details")
	}

	statusBadge := d.statusBadge(row.Status)

	lines := []string{
		d.styles.Title.Render(row.Ingredient.Name),
		d.styles.Muted.Render("Ingredient ID: " + row.Ingredient.ID.String()),
		d.styles.Muted.Render("Inventory ID: " + row.Inventory.ID.String()),
		d.styles.Subtitle.Render("Category: ") + string(row.Ingredient.Category),
		d.styles.Subtitle.Render("Unit: ") + string(row.Ingredient.Unit),
		"",
		d.styles.Subtitle.Render("Quantity: ") + row.Quantity,
		d.styles.Subtitle.Render("Cost per unit: ") + row.Cost,
		d.styles.Subtitle.Render("Status: ") + statusBadge,
	}

	content := strings.Join(lines, "\n")
	if d.width > 0 {
		content = lipgloss.NewStyle().Width(d.width).Render(content)
	}
	return content
}

func (d *DetailViewModel) statusBadge(status string) string {
	switch status {
	case "OUT":
		return components.NewBadge(status, d.styles.ErrorText).View()
	case "LOW":
		return components.NewBadge(status, d.styles.WarningText).View()
	default:
		return components.NewBadge(status, d.styles.Subtitle).View()
	}
}
