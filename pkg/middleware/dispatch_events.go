package middleware

import (
	"fmt"
	"slices"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

// EventDispatcher dispatches domain events to their handlers.
type EventDispatcher interface {
	Dispatch(ctx *Context, event any) error
}

// DispatchEvents dispatches any events collected on the middleware context
// after the command completes.
func DispatchEvents(d EventDispatcher) Middleware {
	return func(ctx *Context, op Operation, next Next) error {
		if op.Kind != OperationKindCommand {
			return next(ctx)
		}

		if err := next(ctx); err != nil {
			return err
		}

		if d == nil {
			return nil
		}

		events := slices.Clone(ctx.Events())
		for _, event := range events {
			if err := d.Dispatch(ctx, event); err != nil {
				var typed *errors.Error
				if errors.As(err, &typed) {
					// A domain handler can veto a mutation with an actionable
					// application error. Preserve its classification for every surface.
					return fmt.Errorf("dispatch event %T: %w", event, err)
				}
				return errors.Internalf("dispatch event %T: %w", event, err)
			}
		}
		return nil
	}
}
