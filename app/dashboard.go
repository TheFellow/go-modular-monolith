package app

import (
	"context"
	"errors"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

const DashboardRecentLimit = 10
const UnknownDashboardCount = -1

type DashboardActivity struct {
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
}

type Dashboard struct {
	DrinkCount      int                 `json:"drink_count"`
	IngredientCount int                 `json:"ingredient_count"`
	InventoryCount  int                 `json:"inventory_count"`
	MenuCount       int                 `json:"menu_count"`
	DraftMenus      int                 `json:"draft_menus"`
	PublishedMenus  int                 `json:"published_menus"`
	LowStockCount   int                 `json:"low_stock_count"`
	OrderCount      int                 `json:"order_count"`
	PendingOrders   int                 `json:"pending_orders"`
	AuditCount      int                 `json:"audit_count"`
	RecentActivity  []DashboardActivity `json:"recent_activity"`
}

func UnknownDashboard() Dashboard {
	return Dashboard{DrinkCount: -1, IngredientCount: -1, InventoryCount: -1, MenuCount: -1, DraftMenus: -1, PublishedMenus: -1, LowStockCount: -1, OrderCount: -1, PendingOrders: -1, AuditCount: -1}
}

// Dashboard returns the read-only application dashboard shared by every
// presentation surface. Counts remain -1 when authorization or another query
// error makes that individual value unavailable; the first error is returned.
func (a *App) Dashboard(ctx *middleware.Context) (Dashboard, error) {
	if a == nil {
		return Dashboard{}, errors.New("dashboard requires an application")
	}
	data := UnknownDashboard()
	var first error
	load := func(target *int, fn func() (int, error)) {
		value, err := fn()
		if err != nil {
			if first == nil && !apperrors.IsPermission(err) {
				first = err
			}
			return
		}
		*target = value
	}
	load(&data.DrinkCount, func() (int, error) { return a.Drinks.Count(ctx, drinks.ListRequest{}) })
	load(&data.IngredientCount, func() (int, error) { return a.Ingredients.Count(ctx, ingredients.ListRequest{}) })
	load(&data.InventoryCount, func() (int, error) { return a.Inventory.Count(ctx, inventory.ListRequest{}) })
	load(&data.LowStockCount, func() (int, error) {
		return a.Inventory.Count(ctx, inventory.ListRequest{LowStock: optional.Some(inventory.DefaultLowStockThreshold)})
	})
	load(&data.MenuCount, func() (int, error) { return a.Menus.Count(ctx, menus.ListRequest{}) })
	load(&data.DraftMenus, func() (int, error) { return a.Menus.Count(ctx, menus.ListRequest{Status: menumodels.MenuStatusDraft}) })
	load(&data.PublishedMenus, func() (int, error) {
		return a.Menus.Count(ctx, menus.ListRequest{Status: menumodels.MenuStatusPublished})
	})
	load(&data.OrderCount, func() (int, error) { return a.Orders.Count(ctx, orders.ListRequest{}) })
	load(&data.PendingOrders, func() (int, error) {
		return a.Orders.Count(ctx, orders.ListRequest{Status: ordersmodels.OrderStatusPending})
	})
	load(&data.AuditCount, func() (int, error) { return a.Audit.Count(ctx, audit.ListRequest{}) })
	page, err := a.Audit.List(ctx, audit.ListRequest{Limit: DashboardRecentLimit})
	if err != nil {
		if first == nil && !apperrors.IsPermission(err) {
			first = err
		}
	} else {
		for _, entry := range page.Items {
			if entry == nil {
				if first == nil {
					first = errors.New("dashboard audit entry is missing")
				}
				continue
			}
			at := entry.CompletedAt
			if at.IsZero() {
				at = entry.StartedAt
			}
			data.RecentActivity = append(data.RecentActivity, DashboardActivity{Timestamp: at, Actor: entry.Principal.String(), Action: entry.Action})
		}
	}
	return data, first
}

func (s *Session) Dashboard() (Dashboard, error) {
	if s == nil || s.App == nil {
		return Dashboard{}, errors.New("dashboard requires an application session")
	}
	return s.App.Dashboard(s.Context())
}

func (s *Session) DashboardContext(ctx context.Context) (Dashboard, error) {
	if s == nil || s.App == nil {
		return Dashboard{}, errors.New("dashboard requires an application session")
	}
	return s.App.Dashboard(s.ContextFrom(ctx))
}
