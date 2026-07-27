package tui

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/mvvm"
)

type orderItem = mvvm.ListItem[models.Order]

func newOrderItem(order models.Order, menuName string, styles tui.ListViewStyles) orderItem {
	status := orderStatusBadge(order.Status, styles)
	description := fmt.Sprintf("%s | %s | %d items", status, menuName, len(order.Items))
	return mvvm.NewListItem(order, truncateID(order.ID.String()), description, order.ID.String())
}

func truncateID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[len(id)-8:]
}
