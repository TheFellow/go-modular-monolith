package middleware

import (
	"time"
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
