package app

import (
	"context"
	"log/slog"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
	"github.com/cedar-policy/cedar-go"
)

type App struct {
	Store       *store.Store
	Audit       *audit.Module
	Drinks      *drinks.Module
	Ingredients *ingredients.Module
	Inventory   *inventory.Module
	Menus       *menus.Module
	Orders      *orders.Module

	logger    *slog.Logger
	metrics   telemetry.Metrics
	principal cedar.EntityUID
}

// New constructs the application around a required store. Domain modules
// register their private persistence models before New returns.
func New(s *store.Store, principal cedar.EntityUID, logger *slog.Logger, metrics telemetry.Metrics) *App {
	a := &App{
		Store:     s,
		logger:    logger,
		metrics:   metrics,
		principal: principal,
	}

	pipeline := middleware.NewPipeline(middleware.PipelineConfig{
		Store:      a.Store,
		Dispatcher: dispatcher.New(s),
		Metrics:    a.metrics,
		RecordActivity: func(ctx *middleware.Context, activity middlewareevents.Activity) error {
			return a.Audit.RecordActivity(ctx, activity)
		},
	})
	a.Audit = audit.NewModule(s, pipeline)
	a.Drinks = drinks.NewModule(s, pipeline)
	a.Ingredients = ingredients.NewModule(s, pipeline)
	a.Inventory = inventory.NewModule(s, pipeline)
	a.Menus = menus.NewModule(s, pipeline)
	a.Orders = orders.NewModule(s, pipeline)

	return a
}

func (a *App) Close() error {
	return a.Store.Close()
}

func (a *App) Context() *middleware.Context {
	return a.ContextFrom(context.Background())
}

func (a *App) ContextFrom(parent context.Context) *middleware.Context {
	return a.contextWithPrincipal(parent, a.principal)
}

func (a *App) ContextFor(parent context.Context, principal cedar.EntityUID) *middleware.Context {
	return a.contextWithPrincipal(parent, principal)
}

func (a *App) contextWithPrincipal(parent context.Context, principal cedar.EntityUID) *middleware.Context {
	if parent == nil {
		parent = context.Background()
	}

	parent = log.ToContext(parent, a.logger.With(log.Actor(principal)))
	parent = telemetry.WithMetrics(parent, a.metrics)

	return middleware.NewContext(parent, principal)
}
