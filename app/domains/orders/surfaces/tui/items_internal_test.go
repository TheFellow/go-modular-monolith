package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
)

func TestOrderItemUsesMenuNameAndCompactMetadata(t *testing.T) {
	t.Parallel()
	order := models.Order{
		ID:        entity.NewOrderID(),
		Status:    models.OrderStatusPending,
		CreatedAt: time.Date(2026, time.August, 30, 18, 40, 0, 0, time.UTC),
		Items:     []models.OrderItem{{Quantity: 2}},
		Tags:      tag.Tags{{Key: "rush"}, {Key: "table", Value: "12"}},
	}
	item := newOrderItem(order, "Dinner", styles.Standard.ListView)
	testutil.Equals(t, item.Title(), "Dinner")
	for _, value := range []string{"Pending", "1 items", "placed 2026-08-30 18:40", "tags: rush,table=12"} {
		testutil.ErrorIf(t, !strings.Contains(item.Description(), value), "description %q missing %q", item.Description(), value)
	}
	testutil.ErrorIf(t, strings.Contains(item.Title(), order.ID.String()), "order id leaked into list title %q", item.Title())
	testutil.ErrorIf(t, !strings.Contains(item.FilterValue(), order.ID.String()), "order id should remain searchable")
}
