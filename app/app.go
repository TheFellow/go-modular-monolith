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
	Tags  *tagging.Repository

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
	auditWriter := audit.NewWriter(s)
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{
		Store:          s,
		Dispatcher:     dispatcher.New(s, tags),
		Metrics:        telemetry.FromContext(ctx),
		RecordActivity: auditWriter.RecordActivity,
	})

	return &App{
		Store:       s,
		Tags:        tags,
		Audit:       audit.NewModule(s, pipeline),
		Drinks:      drinks.NewModule(ctx, s, tags, pipeline),
		Ingredients: ingredients.NewModule(ctx, s, tags, pipeline),
		Inventory:   inventory.NewModule(ctx, s, tags, pipeline),
		Menus:       menus.NewModule(ctx, s, tags, pipeline),
		Orders:      orders.NewModule(ctx, s, tags, pipeline),
	}
}

func (a *App) Close() error {
	return a.Store.Close()
}
