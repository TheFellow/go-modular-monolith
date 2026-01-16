package middleware

import (
	"slices"

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
	return func(ctx *Context, _ cedar.EntityUID, next CommandNext) error {
		if err := next(ctx); err != nil {
			return err
		}

		d, ok := DispatcherFromContext(ctx.Context)
		if !ok || d == nil {
			return nil
		}

		events := slices.Clone(ctx.Events())
		for _, event := range events {
			if err := d.Dispatch(ctx, event); err != nil {
				return errors.Internalf("dispatch event %T: %w", event, err)
			}
		}
		return nil
	}
}
