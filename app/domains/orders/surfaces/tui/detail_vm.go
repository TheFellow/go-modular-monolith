package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	drinksqueries "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	menusmodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	menusqueries "github.com/TheFellow/go-modular-monolith/app/domains/menus/queries"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/charmbracelet/lipgloss"
	"github.com/govalues/decimal"
)

// DetailViewModel renders an order detail pane.
type DetailViewModel struct {
	styles       tui.ListViewStyles
	width        int
	height       int
	order        optional.Value[models.Order]
	ctx          *middleware.Context
	drinkQueries *drinksqueries.Queries
	menuQueries  *menusqueries.Queries
}

func NewDetailViewModel(styles tui.ListViewStyles, ctx *middleware.Context) *DetailViewModel {
	return &DetailViewModel{
		styles:       styles,
		ctx:          ctx,
		drinkQueries: drinksqueries.New(),
		menuQueries:  menusqueries.New(),
	}
}

func (d *DetailViewModel) SetSize(width, height int) {
	d.width = width
	d.height = height
}

func (d *DetailViewModel) SetOrder(order optional.Value[models.Order]) {
	d.order = order
}

func (d *DetailViewModel) View() string {
	order, ok := d.order.Unwrap()
	if !ok {
		return d.styles.Subtitle.Render("Select an order to view details")
	}

	menu, err := d.menu(order.MenuID)
	if err != nil {
		return d.styles.ErrorText.Render(fmt.Sprintf("Error: %v", err))
	}

	statusBadge := orderStatusBadge(order.Status, d.styles)
	lines := []string{
		d.styles.Title.Render("Order"),
		d.styles.Muted.Render("ID: " + order.ID.String()),
		d.styles.Subtitle.Render("Menu: ") + menu.Name,
		d.styles.Subtitle.Render("Status: ") + statusBadge,
		d.styles.Muted.Render("Created: " + formatTime(order.CreatedAt)),
	}

	if completedAt, ok := order.CompletedAt.Unwrap(); ok {
		lines = append(lines, d.styles.Muted.Render("Completed: "+formatTime(completedAt)))
	}

	if strings.TrimSpace(order.Notes) != "" {
		lines = append(lines, "", d.styles.Subtitle.Render("Notes"), order.Notes)
	}

	itemLines, total, err := d.renderItems(order.Items, menu)
	if err != nil {
		lines = append(lines, d.styles.ErrorText.Render(fmt.Sprintf("Error: %v", err)))
	} else {
		lines = append(lines, "", d.styles.Subtitle.Render("Items"))
		lines = append(lines, itemLines...)
		lines = append(lines, "", d.styles.Subtitle.Render("Total: ")+total)
	}

	content := strings.Join(lines, "\n")
	if d.width > 0 {
		content = lipgloss.NewStyle().Width(d.width).Render(content)
	}
	return content
}

func (d *DetailViewModel) renderItems(items []models.OrderItem, menu *menusmodels.Menu) ([]string, string, error) {
	if len(items) == 0 {
		return []string{d.styles.Muted.Render("No items")}, "N/A", nil
	}

	menuItems := make(map[string]menusmodels.MenuItem, len(menu.Items))
	for _, item := range menu.Items {
		menuItems[item.DrinkID.String()] = item
	}

	sorted := make([]models.OrderItem, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].DrinkID.String() < sorted[j].DrinkID.String() })

	lines := make([]string, 0, len(sorted))
	var total menusmodels.Price
	var totalSet bool
	totalAvailable := true

	for _, item := range sorted {
		name, err := d.drinkName(item.DrinkID)
		if err != nil {
			return nil, "", err
		}

		lineTotal := "N/A"
		if menuItem, ok := menuItems[item.DrinkID.String()]; ok {
			if price, ok := menuItem.Price.Unwrap(); ok {
				qty, err := decimal.New(int64(item.Quantity), 0)
				if err != nil {
					return nil, "", errors.Internalf("quantity %d: %w", item.Quantity, err)
				}
				linePrice, err := price.Mul(qty)
				if err != nil {
					return nil, "", err
				}
				lineTotal = linePrice.String()
				if totalAvailable {
					if !totalSet {
						total = linePrice
						totalSet = true
					} else {
						next, err := total.Add(linePrice)
						if err != nil {
							return nil, "", err
						}
						total = next
					}
				}
			} else {
				totalAvailable = false
			}
		} else {
			totalAvailable = false
		}

		lines = append(lines, fmt.Sprintf("- %s | qty: %d | total: %s", name, item.Quantity, lineTotal))
	}

	totalStr := "N/A"
	if totalAvailable && totalSet {
		totalStr = total.String()
	}
	return lines, totalStr, nil
}

func (d *DetailViewModel) drinkName(id entity.DrinkID) (string, error) {
	if id.IsZero() {
		return "", errors.Internalf("order item missing drink id")
	}
	item, err := d.drinkQueries.Get(d.ctx, id)
	if err != nil {
		return "", errors.Internalf("load drink %s: %w", id.String(), err)
	}
	if item == nil {
		return "", errors.Internalf("drink %s missing", id.String())
	}
	name := strings.TrimSpace(item.Name)
	if name == "" {
		return "", errors.Internalf("drink %s missing name", id.String())
	}
	return name, nil
}

func (d *DetailViewModel) menu(id entity.MenuID) (*menusmodels.Menu, error) {
	if id.IsZero() {
		return nil, errors.Internalf("order missing menu id")
	}
	menu, err := d.menuQueries.Get(d.ctx, id)
	if err != nil {
		return nil, errors.Internalf("load menu %s: %w", id.String(), err)
	}
	if menu == nil {
		return nil, errors.Internalf("menu %s missing", id.String())
	}
	if strings.TrimSpace(menu.Name) == "" {
		return nil, errors.Internalf("menu %s missing name", id.String())
	}
	return menu, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
