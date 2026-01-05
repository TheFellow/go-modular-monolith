package middleware

import (
	"log"

	cedar "github.com/cedar-policy/cedar-go"
)

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

		for _, event := range ctx.Events() {
			if err := d.Dispatch(ctx, event); err != nil {
				log.Printf("handler error for %T: %v", event, err)
			}
		}
		return nil
	}
}
