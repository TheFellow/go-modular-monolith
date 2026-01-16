package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/cedar-policy/cedar-go"

	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
)

func TrackActivity() CommandMiddleware {
	return func(ctx *Context, action cedar.EntityUID, next CommandNext) error {
		activity := middlewareevents.NewActivity(action, cedar.EntityUID{}, ctx.Principal())
		WithActivity(activity)(ctx)

		err := next(ctx)

		if activity.Resource.IsZero() {
			if len(activity.Touches) > 0 {
				activity.Resource = activity.Touches[0]
			}
		}
		activity.Complete(err)

		d, ok := DispatcherFromContext(ctx.Context)
		if ok && d != nil {
			event := middlewareevents.ActivityCompleted{Activity: *activity}
			if derr := d.Dispatch(ctx, event); derr != nil {
				log.FromContext(ctx).Error("dispatch activity completed", log.Err(derr))
				if err == nil {
					return errors.Internalf("dispatch activity completed: %w", derr)
				}
			}
		}

		return err
	}
}
