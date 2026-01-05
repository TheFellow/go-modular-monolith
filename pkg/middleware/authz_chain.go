package middleware

import (
	"log/slog"
	"time"

	cedar "github.com/cedar-policy/cedar-go"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
)

// AuthZNext is the continuation function for AuthZ middleware.
type AuthZNext func() error

// AuthZMiddleware wraps an authorization check with observability.
type AuthZMiddleware func(ctx *Context, action cedar.EntityUID, next AuthZNext) error

// AuthZChain executes a sequence of AuthZ middleware.
type AuthZChain struct {
	middlewares []AuthZMiddleware
}

// NewAuthZChain creates a new AuthZ middleware chain.
func NewAuthZChain(middlewares ...AuthZMiddleware) *AuthZChain {
	return &AuthZChain{middlewares: middlewares}
}

// Execute runs the AuthZ chain with the given final authorization function.
func (c *AuthZChain) Execute(ctx *Context, action cedar.EntityUID, final AuthZNext) error {
	next := final
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		m := c.middlewares[i]
		prev := next
		next = func() error {
			return m(ctx, action, prev)
		}
	}
	return next()
}

// AuthZLogging logs authorization decisions with duration.
func AuthZLogging() AuthZMiddleware {
	return func(ctx *Context, _ cedar.EntityUID, next AuthZNext) error {
		logger := log.FromContext(ctx)
		start := time.Now()

		err := next()
		duration := time.Since(start)

		if err != nil {
			if errors.IsPermission(err) {
				logger.Info("authorization denied",
					log.Allowed(false),
					slog.Duration("duration", duration),
				)
			} else {
				logger.Warn("authorization error",
					log.Allowed(false),
					slog.Duration("duration", duration),
					log.Err(err),
				)
			}
			return err
		}

		logger.Debug("authorization allowed",
			log.Allowed(true),
			slog.Duration("duration", duration),
		)
		return nil
	}
}

// AuthZMetrics records authorization metrics (latency and decisions).
func AuthZMetrics() AuthZMiddleware {
	return func(ctx *Context, action cedar.EntityUID, next AuthZNext) error {
		m := telemetry.FromContext(ctx)
		actionLabel := actionLabel(action)
		start := time.Now()

		err := next()

		m.Histogram(telemetry.MetricAuthZLatency, telemetry.LabelAction).
			ObserveDuration(start, actionLabel)

		switch {
		case err == nil:
			m.Counter(telemetry.MetricAuthZTotal, telemetry.LabelAction, telemetry.LabelDecision).
				Inc(actionLabel, "allow")
		case errors.IsPermission(err):
			m.Counter(telemetry.MetricAuthZTotal, telemetry.LabelAction, telemetry.LabelDecision).
				Inc(actionLabel, "deny")
			m.Counter(telemetry.MetricAuthZDenied, telemetry.LabelAction).
				Inc(actionLabel)
		default:
			m.Counter(telemetry.MetricAuthZTotal, telemetry.LabelAction, telemetry.LabelDecision).
				Inc(actionLabel, "error")
		}

		return err
	}
}

// DefaultAuthZChain is the standard AuthZ chain with logging and metrics.
var DefaultAuthZChain = NewAuthZChain(
	AuthZLogging(),
	AuthZMetrics(),
)
