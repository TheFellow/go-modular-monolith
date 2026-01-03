package middleware

import (
	"log"

	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	cedar "github.com/cedar-policy/cedar-go"
)

func Dispatcher(d *dispatcher.Dispatcher) CommandMiddleware {
	return func(ctx *Context, _ cedar.EntityUID, _ cedar.Entity, next CommandNext) error {
		if err := next(ctx); err != nil {
			return err
		}

		for _, event := range ctx.Events() {
			if err := d.Dispatch(ctx, event); err != nil {
				log.Printf("handler error for %T: %v", event, err)
			}
		}
		return nil
	}
}
