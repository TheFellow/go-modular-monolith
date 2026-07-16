package app

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
)

type App struct {
	Store       *store.Store
	Audit       *audit.Module
	Drinks      *drinks.Module
	Ingredients *ingredients.Module
	Inventory   *inventory.Module
	Menus       *menus.Module
	Orders      *orders.Module

	ctx context.Context
}

// New constructs the application around a required store. Domain modules
// register their private persistence models before New returns.
func New(ctx context.Context, config Config) *App {
	s := config.Store
	a := &App{
		Store: s,
		ctx:   ctx,
	}

	pipeline := middleware.NewPipeline(middleware.PipelineConfig{
		Store:      a.Store,
		Dispatcher: dispatcher.New(s),
		Metrics:    telemetry.FromContext(ctx),
		RecordActivity: func(ctx *middleware.Context, activity middlewareevents.Activity) error {
			return a.Audit.RecordActivity(ctx, activity)
		},
	})
	a.Audit = audit.NewModule(ctx, s, pipeline)
	a.Drinks = drinks.NewModule(ctx, s, pipeline)
	a.Ingredients = ingredients.NewModule(ctx, s, pipeline)
	a.Inventory = inventory.NewModule(ctx, s, pipeline)
	a.Menus = menus.NewModule(ctx, s, pipeline)
	a.Orders = orders.NewModule(ctx, s, pipeline)

	return a
}

func (a *App) Close() error {
	return a.Store.Close()
}

func (a *App) Context() *middleware.Context {
	return a.ContextFrom(a.ctx)
}

func (a *App) ContextFrom(parent context.Context) *middleware.Context {
	principal := authn.FromContext(parent)
	parent = log.ToContext(parent, log.FromContext(parent).With(log.Actor(principal)))

	return middleware.NewContext(parent, principal)
}
