package tui

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
)

type orderItem = tui.ListItem[models.Order]

func newOrderItem(order models.Order, menuName string, styles tui.ListViewStyles) orderItem {
	status := orderStatusBadge(order.Status, styles)
	placed := ""
	if !order.CreatedAt.IsZero() {
		placed = "placed " + order.CreatedAt.Format("2006-01-02 15:04")
	}
	tags := order.Tags.Canonical().String()
	description := tui.ListSummary(status, fmt.Sprintf("%d items", len(order.Items)), placed, tui.TagSummary(tags))
	filterValue := tui.ListSummary(menuName, string(order.Status), tags, order.ID.String())
	return tui.NewListItem(order, menuName, description, filterValue)
}

func truncateID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[len(id)-8:]
}
