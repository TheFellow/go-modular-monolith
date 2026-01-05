package middleware

import (
	cedar "github.com/cedar-policy/cedar-go"
)

// EventDispatcher dispatches domain events to their handlers.
type EventDispatcher interface {
	Dispatch(ctx *Context, event any) error
}

// DispatchEvents dispatches any events collected on the middleware context
// after the command completes. Each event may be dispatched through optional
// event middleware (e.g., metrics) passed explicitly by the caller.
func DispatchEvents(eventMiddlewares ...EventMiddleware) CommandMiddleware {
	eventChain := NewEventChain(eventMiddlewares...)
	return func(ctx *Context, _ cedar.EntityUID, _ cedar.Entity, next CommandNext) error {
		if err := next(ctx); err != nil {
			return err
		}

		d, ok := DispatcherFromContext(ctx.Context)
		if !ok || d == nil {
			return nil
		}

		for _, event := range ctx.Events() {
			if err := eventChain.Execute(ctx, event, func() error {
				return d.Dispatch(ctx, event)
			}); err != nil {
				return err
			}
		}
		return nil
	}
}
