package app

import (
	"context"
	"log/slog"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
	"github.com/cedar-policy/cedar-go"
)

type App struct {
	Store       optional.Value[*store.Store]
	Dispatcher  middleware.EventDispatcher
	Logger      *slog.Logger
	Metrics     telemetry.Metrics
	Audit       *audit.Module
	Drinks      *drinks.Module
	Ingredients *ingredients.Module
	Inventory   *inventory.Module
	Menu        *menu.Module
	Orders      *orders.Module

	metricsCollector *middleware.MetricsCollector
}

func New(opts ...Option) *App {
	a := &App{
		Store:       optional.None[*store.Store](),
		Dispatcher:  dispatcher.New(),
		Logger:      slog.Default(),
		Metrics:     telemetry.Nop(),
		Audit:       audit.NewModule(),
		Drinks:      drinks.NewModule(),
		Ingredients: ingredients.NewModule(),
		Inventory:   inventory.NewModule(),
		Menu:        menu.NewModule(),
		Orders:      orders.NewModule(),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(a)
		}
	}

	if a.Logger == nil {
		a.Logger = slog.Default()
	}
	if a.Metrics == nil {
		a.Metrics = telemetry.Nop()
	}
	a.metricsCollector = middleware.NewMetricsCollector(a.Metrics)

	return a
}

func (a *App) Close() error {
	if a == nil {
		return nil
	}
	s, ok := a.Store.Unwrap()
	if !ok || s == nil {
		return nil
	}
	return s.Close()
}

func (a *App) Context(parent context.Context, principal cedar.EntityUID) *middleware.Context {
	if parent == nil {
		parent = context.Background()
	}

	if a != nil {
		parent = log.ToContext(parent, a.Logger.With(log.Actor(principal)))
		parent = telemetry.WithMetrics(parent, a.Metrics)
	}

	opts := []middleware.ContextOpt{
		middleware.WithPrincipal(principal),
	}
	if a != nil {
		if s, ok := a.Store.Unwrap(); ok && s != nil {
			opts = append(opts, middleware.WithStore(s))
		}
		if a.Dispatcher != nil {
			opts = append(opts, middleware.WithEventDispatcher(a.Dispatcher))
		}
		if a.metricsCollector != nil {
			opts = append(opts, middleware.WithMetricsCollector(a.metricsCollector))
		}
	}
	return middleware.NewContext(parent, opts...)
}
