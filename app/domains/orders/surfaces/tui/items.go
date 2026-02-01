package tui

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
)

type orderItem struct {
	order       models.Order
	menuName    string
	description string
	displayID   string
}

func newOrderItem(order models.Order, menuName string, styles ListViewStyles) orderItem {
	status := orderStatusBadge(order.Status, styles)
	description := fmt.Sprintf("%s | %s | %d items", status, menuName, len(order.Items))
	return orderItem{
		order:       order,
		menuName:    menuName,
		description: description,
		displayID:   truncateID(order.ID.String()),
	}
}

func (i orderItem) Title() string { return i.displayID }
func (i orderItem) Description() string {
	return i.description
}
func (i orderItem) FilterValue() string { return i.order.ID.String() }

func truncateID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[len(id)-8:]
}
