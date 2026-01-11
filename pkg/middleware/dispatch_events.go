package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/cedar-policy/cedar-go"
)

// EventDispatcher dispatches domain events to their handlers.
type EventDispatcher interface {
	Dispatch(ctx *Context, event any) error
}

// DispatchEvents dispatches any events collected on the middleware context
// after the command completes.
func DispatchEvents() CommandMiddleware {
	return func(ctx *Context, _ cedar.EntityUID, _ cedar.Entity, next CommandNext) error {
		if err := next(ctx); err != nil {
			return err
		}

		d, ok := DispatcherFromContext(ctx.Context)
		if !ok || d == nil {
			return nil
		}

		for i := 0; ; i++ {
			events := ctx.Events()
			if i >= len(events) {
				break
			}
			if err := d.Dispatch(ctx, events[i]); err != nil {
				return errors.Internalf("dispatch event %T: %w", events[i], err)
			}
		}
		return nil
	}
}
