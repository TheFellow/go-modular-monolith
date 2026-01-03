package middleware

import (
	"log"

	cedar "github.com/cedar-policy/cedar-go"
)

type EventDispatcher interface {
	Dispatch(ctx *Context, event any) error
}

func Dispatcher(d EventDispatcher) CommandMiddleware {
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
