package app

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
)

type App struct {
	Store *store.Store
	Tags  *tagging.Module

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
	audit.RegisterSchema(ctx, s)
	tagging.RegisterSchema(ctx, s)
	tags := tagging.NewRepository(s)
	targets := tagging.NewRegistry()
	auditWriter := audit.NewWriter(s)
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{
		Store:          s,
		Dispatcher:     dispatcher.New(s, tags),
		Metrics:        telemetry.FromContext(ctx),
		RecordActivity: auditWriter.RecordActivity,
	})

	drinksModule := drinks.NewModule(ctx, s, tags, targets, pipeline)
	ingredientsModule := ingredients.NewModule(ctx, s, tags, targets, pipeline)
	inventoryModule := inventory.NewModule(ctx, s, tags, targets, pipeline)
	menusModule := menus.NewModule(ctx, s, tags, targets, pipeline)
	ordersModule := orders.NewModule(ctx, s, tags, targets, pipeline)

	return &App{
		Store:       s,
		Tags:        tagging.NewModule(tags, targets, pipeline),
		Audit:       audit.NewModule(s, pipeline),
		Drinks:      drinksModule,
		Ingredients: ingredientsModule,
		Inventory:   inventoryModule,
		Menus:       menusModule,
		Orders:      ordersModule,
	}
}

func (a *App) Close() error {
	return a.Store.Close()
}
