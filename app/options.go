package app

import (
	"log/slog"

	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
	"github.com/cedar-policy/cedar-go"
)

type Option func(*App)

func WithStore(s *store.Store) Option {
	return func(a *App) {
		if a == nil {
			return
		}
		if s == nil {
			a.Store = optional.None[*store.Store]()
			return
		}
		a.Store = optional.Some(s)
	}
}

func WithLogger(l *slog.Logger) Option {
	return func(a *App) {
		if a == nil {
			return
		}
		a.Logger = l
	}
}

func WithMetrics(m telemetry.Metrics) Option {
	return func(a *App) {
		if a == nil {
			return
		}
		a.Metrics = m
	}
}

func WithPrincipal(principal cedar.EntityUID) Option {
	return func(a *App) {
		if a == nil {
			return
		}
		a.principal = optional.Some(principal)
	}
}
