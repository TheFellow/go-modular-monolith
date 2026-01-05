package middleware

import (
	"log/slog"
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/log"
)

// EventNext is the continuation function for event middleware.
type EventNext func() error

// EventMiddleware wraps an event dispatch with observability.
type EventMiddleware func(ctx *Context, event any, next EventNext) error

// EventChain executes a sequence of event middleware.
type EventChain struct {
	middlewares []EventMiddleware
}

// NewEventChain creates a new event middleware chain.
func NewEventChain(middlewares ...EventMiddleware) *EventChain {
	return &EventChain{middlewares: middlewares}
}

// Execute runs the event chain with the given final dispatch function.
func (c *EventChain) Execute(ctx *Context, event any, final EventNext) error {
	next := final
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		m := c.middlewares[i]
		prev := next
		next = func() error {
			return m(ctx, event, prev)
		}
	}
	return next()
}

// EventLogging logs event dispatch with duration.
// Sets the event_type attribute in the log context.
func EventLogging() EventMiddleware {
	return func(ctx *Context, event any, next EventNext) error {
		eventType := eventTypeLabel(event)

		// Add event_type to log context
		base := ctx.Context
		ctx.Context = log.WithLogAttrs(base, log.EventType(eventType))
		defer func() { ctx.Context = base }()

		logger := log.FromContext(ctx)
		start := time.Now()

		logger.Debug("dispatching event")

		err := next()
		duration := time.Since(start)

		if err != nil {
			logger.Error("event handler failed",
				slog.Duration("duration", duration),
				log.Err(err),
			)
			return err
		}

		logger.Debug("event dispatched",
			slog.Duration("duration", duration),
		)
		return nil
	}
}

// EventMetrics records event dispatch metrics (count, latency, errors).
func EventMetrics() EventMiddleware {
	return func(ctx *Context, event any, next EventNext) error {
		mc, ok := MetricsCollectorFromContext(ctx.Context)
		if !ok || mc == nil {
			mc = nopMetricsCollector
		}

		start := time.Now()

		err := next()

		mc.RecordEvent(event, time.Since(start), err)
		return err
	}
}

// DefaultEventChain is the standard event chain with logging and metrics.
var DefaultEventChain = NewEventChain(
	EventLogging(),
	EventMetrics(),
)
