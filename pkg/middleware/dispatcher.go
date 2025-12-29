package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	cedar "github.com/cedar-policy/cedar-go"
)

func Dispatcher(d *dispatcher.Dispatcher) CommandMiddleware {
	return func(ctx *Context, _ cedar.EntityUID, _ cedar.Entity, next CommandNext) error {
		if err := next(ctx); err != nil {
			return err
		}
		d.Dispatch(ctx.Events())
		return nil
	}
}
