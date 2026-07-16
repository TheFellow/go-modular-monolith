package app

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
)

type App struct {
	Store *store.Store

	Audit       *audit.Module
	Drinks      *drinks.Module
	Ingredients *ingredients.Module
	Inventory   *inventory.Module
	Menus       *menus.Module
	Orders      *orders.Module
}

// New constructs the application around a required store. Domain modules
// register their private persistence models before New returns.
func New(ctx context.Context, config Config) *App {
	s := config.Store
	auditWriter := audit.NewWriter(ctx, s)
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{
		Store:          s,
		Dispatcher:     dispatcher.New(s),
		Metrics:        telemetry.FromContext(ctx),
		RecordActivity: auditWriter.RecordActivity,
	})

	return &App{
		Store:       s,
		Audit:       audit.NewModule(s, pipeline),
		Drinks:      drinks.NewModule(ctx, s, pipeline),
		Ingredients: ingredients.NewModule(ctx, s, pipeline),
		Inventory:   inventory.NewModule(ctx, s, pipeline),
		Menus:       menus.NewModule(ctx, s, pipeline),
		Orders:      orders.NewModule(ctx, s, pipeline),
	}
}

func (a *App) Close() error {
	return a.Store.Close()
}
