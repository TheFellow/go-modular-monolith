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
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
	"github.com/cedar-policy/cedar-go"
)

type App struct {
	Store       *store.Store
	Dispatcher  middleware.EventDispatcher
	Logger      *slog.Logger
	Metrics     telemetry.Metrics
	Audit       *audit.Module
	Drinks      *drinks.Module
	Ingredients *ingredients.Module
	Inventory   *inventory.Module
	Menus       *menus.Module
	Orders      *orders.Module

	principal optional.Value[cedar.EntityUID]
	pipeline  *middleware.Pipeline
}

type activityRecorder struct {
	app *App
}

func (r activityRecorder) RecordActivity(ctx *middleware.Context, activity middlewareevents.Activity) error {
	return r.app.Audit.RecordActivity(ctx, activity)
}

// New constructs the application around a required store. Domain modules
// register their private persistence models before New returns.
func New(s *store.Store, opts ...Option) *App {
	a := &App{
		Store:      s,
		Dispatcher: dispatcher.New(s),
		Logger:     slog.Default(),
		Metrics:    telemetry.Nop(),
		principal:  optional.None[cedar.EntityUID](),
	}

	for _, opt := range opts {
		opt(a)
	}

	a.pipeline = middleware.NewPipeline(middleware.PipelineConfig{
		Store:            a.Store,
		Dispatcher:       a.Dispatcher,
		Metrics:          a.Metrics,
		ActivityRecorder: activityRecorder{app: a},
	})
	a.Audit = audit.NewModule(s, a.pipeline)
	a.Drinks = drinks.NewModule(s, a.pipeline)
	a.Ingredients = ingredients.NewModule(s, a.pipeline)
	a.Inventory = inventory.NewModule(s, a.pipeline)
	a.Menus = menus.NewModule(s, a.pipeline)
	a.Orders = orders.NewModule(s, a.pipeline)

	return a
}

func (a *App) Close() error {
	return a.Store.Close()
}

func (a *App) Context() *middleware.Context {
	return a.ContextFrom(context.Background())
}

func (a *App) ContextFrom(parent context.Context) *middleware.Context {
	return a.contextWithPrincipal(parent, a.principalOrAnonymous())
}

func (a *App) ContextFor(parent context.Context, principal cedar.EntityUID) *middleware.Context {
	return a.contextWithPrincipal(parent, principal)
}

func (a *App) principalOrAnonymous() cedar.EntityUID {
	if principal, ok := a.principal.Unwrap(); ok {
		return principal
	}
	return authn.Anonymous()
}

func (a *App) contextWithPrincipal(parent context.Context, principal cedar.EntityUID) *middleware.Context {
	if parent == nil {
		parent = context.Background()
	}

	parent = log.ToContext(parent, a.Logger.With(log.Actor(principal)))
	parent = telemetry.WithMetrics(parent, a.Metrics)

	return middleware.NewContext(parent, principal)
}
