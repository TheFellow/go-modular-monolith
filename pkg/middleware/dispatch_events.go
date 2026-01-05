package middleware

import (
	"log/slog"
	"time"

	cedar "github.com/cedar-policy/cedar-go"

	"github.com/TheFellow/go-modular-monolith/pkg/log"
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

		mc, _ := MetricsCollectorFromContext(ctx.Context)

		for _, event := range ctx.Events() {
			base := ctx.Context
			eventType := eventTypeLabel(event)
			ctx.Context = log.WithLogAttrs(base, log.EventType(eventType))

			logger := log.FromContext(ctx)
			start := time.Now()
			logger.Debug("dispatching event")
			if err := d.Dispatch(ctx, event); err != nil {
				logger.Error("event handler failed", slog.Duration("duration", time.Since(start)), log.Err(err))
				if mc != nil {
					mc.RecordEvent(event, time.Since(start), err)
				}
				return err
			}
			logger.Debug("event dispatched", slog.Duration("duration", time.Since(start)))
			if mc != nil {
				mc.RecordEvent(event, time.Since(start), nil)
			}
		}
		return nil
	}
}
