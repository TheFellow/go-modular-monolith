package tui

import (
	"cmp"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/TheFellow/go-modular-monolith/app"
	menus "github.com/TheFellow/go-modular-monolith/app/domains/menus"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
	"github.com/charmbracelet/lipgloss"
)

// DetailViewModel renders a menu detail pane.
type DetailViewModel struct {
	styles       tui.ListViewStyles
	width        int
	height       int
	menu         optional.Value[models.Menu]
	app          *app.Session
	actions      map[actions.ID]actions.State
	readiness    *models.ReadinessReport
	readinessErr error
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

func (d *DetailViewModel) SetMenu(menu optional.Value[models.Menu]) {
	d.menu = menu
}

func (d *DetailViewModel) SetActions(states map[actions.ID]actions.State) { d.actions = states }
func (d *DetailViewModel) SetReadiness(report *models.ReadinessReport, err error) {
	d.readiness, d.readinessErr = report, err
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
		d.styles.Muted.Render("Created: " + formatMenuTime(menu.CreatedAt)),
		d.styles.Subtitle.Render("Status: ") + statusBadge,
		d.styles.Subtitle.Render("Tags: ") + cmp.Or(menu.Tags.Canonical().String(), "(none)"),
	}
	if publishedAt, ok := menu.PublishedAt.Unwrap(); ok {
		lines = append(lines, d.styles.Muted.Render("Published: "+formatMenuTime(publishedAt)))
	}
	if d.readinessErr != nil {
		lines = append(lines, d.styles.ErrorText.Render("Readiness: "+d.readinessErr.Error()))
	} else if d.readiness != nil {
		if len(d.readiness.Findings) == 0 {
			lines = append(lines, d.styles.Subtitle.Render("Readiness: ready"))
		} else {
			lines = append(lines, "", d.styles.Subtitle.Render("Readiness"))
			for _, finding := range d.readiness.Findings {
				lines = append(lines, d.styles.Muted.Render(fmt.Sprintf("- %s: %s", finding.Severity, finding.Message)))
			}
		}
	}

	if strings.TrimSpace(menu.Description) != "" {
		lines = append(lines, "", d.styles.Subtitle.Render("Description"), menu.Description)
	}

	var unavailable []string
	for _, action := range []struct {
		id    actions.ID
		label string
	}{
		{menus.ControlEdit, "Edit"}, {menus.ControlDelete, "Delete"},
		{menus.ControlAddDrink, "Add drink"}, {menus.ControlRemoveDrink, "Remove drink"},
		{menus.ControlPublish, "Publish"}, {menus.ControlDraft, "Return to draft"},
	} {
		if state, ok := d.actions[action.id]; ok && state.Visible && !state.Enabled && state.DisabledReason != "" {
			unavailable = append(unavailable, action.label+": "+state.DisabledReason)
		}
	}
	if len(unavailable) > 0 {
		lines = append(lines, "", d.styles.Subtitle.Render("Unavailable actions"))
		for _, reason := range unavailable {
			lines = append(lines, d.styles.Muted.Render("- "+reason))
		}
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

	parts := []string{name, "Drink ID: " + item.DrinkID.String(), fmt.Sprintf("Sort order: %d", item.SortOrder), menuAvailabilityLabel(item.Availability)}
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

func formatMenuTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}

func (d *DetailViewModel) itemName(item models.MenuItem) (string, error) {
	if name, ok := item.DisplayName.Unwrap(); ok {
		name = strings.TrimSpace(name)
		if name != "" {
			return name, nil
		}
	}

	drink, err := d.app.Drinks.Get(d.context(), item.DrinkID)
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
