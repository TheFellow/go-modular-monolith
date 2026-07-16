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

	ctx       context.Context
	principal cedar.EntityUID
}

// New constructs the application around a required store. Domain modules
// register their private persistence models before New returns.
func New(ctx context.Context, s *store.Store, principal cedar.EntityUID) *App {
	a := &App{
		Store:     s,
		ctx:       ctx,
		principal: principal,
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
	return a.contextWithPrincipal(a.ctx, a.principal)
}

func (a *App) ContextFor(principal cedar.EntityUID) *middleware.Context {
	return a.contextWithPrincipal(a.ctx, principal)
}

func (a *App) contextWithPrincipal(parent context.Context, principal cedar.EntityUID) *middleware.Context {
	parent = log.ToContext(parent, log.FromContext(parent).With(log.Actor(principal)))

	return middleware.NewContext(parent, principal)
}
