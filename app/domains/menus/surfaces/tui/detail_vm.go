package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	drinksqueries "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/charmbracelet/lipgloss"
)

// DetailViewModel renders a menu detail pane.
type DetailViewModel struct {
	styles  tui.ListViewStyles
	width   int
	height  int
	menu    optional.Value[models.Menu]
	app     *app.App
	queries *drinksqueries.Queries
}

func NewDetailViewModel(styles tui.ListViewStyles, app *app.App) *DetailViewModel {
	return &DetailViewModel{
		styles:  styles,
		app:     app,
		queries: drinksqueries.New(),
	}
}

func (d *DetailViewModel) SetSize(width, height int) {
	d.width = width
	d.height = height
}

func (d *DetailViewModel) SetMenu(menu optional.Value[models.Menu]) {
	d.menu = menu
}

func (d *DetailViewModel) View() string {
	menu, ok := d.menu.Unwrap()
	if !ok {
		return d.styles.Subtitle.Render("Select a menu to view details")
	}

	statusBadge := menuStatusBadge(menu.Status, d.styles)
	lines := []string{
		d.styles.Title.Render(menu.Name),
		d.styles.Muted.Render("ID: " + menu.ID.String()),
		d.styles.Subtitle.Render("Status: ") + statusBadge,
	}

	if strings.TrimSpace(menu.Description) != "" {
		lines = append(lines, "", d.styles.Subtitle.Render("Description"), menu.Description)
	}

	lines = append(lines, "", d.styles.Subtitle.Render("Drinks: ")+fmt.Sprintf("%d", len(menu.Items)))

	itemLines, err := d.renderItems(menu.Items)
	if err != nil {
		lines = append(lines, d.styles.ErrorText.Render(fmt.Sprintf("Error: %v", err)))
	} else {
		lines = append(lines, "", d.styles.Subtitle.Render("Menu Items"))
		lines = append(lines, itemLines...)
	}

	content := strings.Join(lines, "\n")
	if d.width > 0 {
		content = lipgloss.NewStyle().Width(d.width).Render(content)
	}
	return content
}

func (d *DetailViewModel) renderItems(items []models.MenuItem) ([]string, error) {
	if len(items) == 0 {
		return []string{d.styles.Muted.Render("No drinks added")}, nil
	}

	sorted := make([]models.MenuItem, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].SortOrder < sorted[j].SortOrder })

	lines := make([]string, 0, len(sorted))
	for _, item := range sorted {
		line, err := d.itemLine(item)
		if err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	return lines, nil
}

func (d *DetailViewModel) itemLine(item models.MenuItem) (string, error) {
	name, err := d.itemName(item)
	if err != nil {
		return "", err
	}

	parts := []string{name, menuAvailabilityLabel(item.Availability)}
	if price, ok := item.Price.Unwrap(); ok {
		parts = append(parts, price.String())
	} else {
		parts = append(parts, "N/A")
	}
	if item.Featured {
		parts = append(parts, "featured")
	}
	return "- " + strings.Join(parts, " | "), nil
}

func (d *DetailViewModel) itemName(item models.MenuItem) (string, error) {
	if name, ok := item.DisplayName.Unwrap(); ok {
		name = strings.TrimSpace(name)
		if name != "" {
			return name, nil
		}
	}

	drink, err := d.queries.Get(d.context(), item.DrinkID)
	if err != nil {
		return "", errors.Internalf("load drink %s: %w", item.DrinkID.String(), err)
	}
	if drink == nil {
		return "", errors.Internalf("drink %s missing", item.DrinkID.String())
	}
	name := strings.TrimSpace(drink.Name)
	if name == "" {
		return "", errors.Internalf("drink %s missing name", item.DrinkID.String())
	}
	return name, nil
}

func (d *DetailViewModel) context() *middleware.Context {
	return d.app.Context()
}

func menuAvailabilityLabel(avail models.Availability) string {
	switch avail {
	case models.AvailabilityAvailable:
		return "Available"
	case models.AvailabilityLimited:
		return "Limited"
	case models.AvailabilityUnavailable:
		return "Unavailable"
	default:
		return titleCase(string(avail))
	}
}
