package middleware

import (
	cedar "github.com/cedar-policy/cedar-go"
)

// EventDispatcher dispatches domain events to their handlers.
type EventDispatcher interface {
	Dispatch(ctx *Context, event any) error
}

// DispatchEvents dispatches any events collected on the middleware context
// after the command completes. Each event is dispatched through the event
// chain which handles logging and metrics.
func DispatchEvents() CommandMiddleware {
	return func(ctx *Context, _ cedar.EntityUID, _ cedar.Entity, next CommandNext) error {
		if err := next(ctx); err != nil {
			return err
		}

		d, ok := DispatcherFromContext(ctx.Context)
		if !ok || d == nil {
			return nil
		}

		for _, event := range ctx.Events() {
			err := DefaultEventChain.Execute(ctx, event, func() error {
				return d.Dispatch(ctx, event)
			})
			if err != nil {
				return err
			}
		}
		return nil
	}
}
