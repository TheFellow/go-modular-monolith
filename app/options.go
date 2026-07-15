package app

import (
	"log/slog"

	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
	"github.com/cedar-policy/cedar-go"
)

type Option func(*App)

func WithLogger(l *slog.Logger) Option {
	return func(a *App) {
		a.Logger = l
	}
}

func WithMetrics(m telemetry.Metrics) Option {
	return func(a *App) {
		a.Metrics = m
	}
}

func WithPrincipal(principal cedar.EntityUID) Option {
	return func(a *App) {
		a.principal = optional.Some(principal)
	}
}
