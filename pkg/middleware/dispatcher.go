package middleware

import (
	"time"

	cedar "github.com/cedar-policy/cedar-go"

	"github.com/TheFellow/go-modular-monolith/pkg/log"
)

type EventDispatcher interface {
	Dispatch(ctx *Context, event any) error
}

func Dispatcher(d EventDispatcher) CommandMiddleware {
	return func(ctx *Context, _ cedar.EntityUID, _ cedar.Entity, next CommandNext) error {
		if err := next(ctx); err != nil {
			return err
		}

		mc, _ := MetricsCollectorFromContext(ctx.Context)

		for _, event := range ctx.Events() {
			start := time.Now()
			log.FromContext(ctx).Debug("dispatching event", log.Args(log.EventType(eventTypeLabel(event)))...)
			if err := d.Dispatch(ctx, event); err != nil {
				if mc != nil {
					mc.RecordEvent(event, time.Since(start), err)
				}
				return err
			}
			if mc != nil {
				mc.RecordEvent(event, time.Since(start), nil)
			}
		}
		return nil
	}
}
