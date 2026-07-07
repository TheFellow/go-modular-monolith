package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"

	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
)

func TrackActivity() Middleware {
	return func(ctx *Context, op Operation, next Next) error {
		if op.Kind != OperationKindCommand {
			return next(ctx)
		}

		activity := middlewareevents.NewActivity(op.Action, op.Resource.UID, ctx.Principal())
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
			dispatch := func(dispatchCtx *Context) error {
				if derr := d.Dispatch(dispatchCtx, event); derr != nil {
					log.FromContext(dispatchCtx).Error("dispatch activity completed", log.Err(derr))
					if err == nil {
						return errors.Internalf("dispatch activity completed: %w", derr)
					}
				}
				return nil
			}

			if tx, ok := ctx.Transaction(); ok && tx != nil {
				if derr := dispatch(ctx); derr != nil {
					return derr
				}
			} else if s, ok := store.FromContext(ctx.Context); ok && s != nil {
				if derr := s.Write(ctx, func(tx *bstore.Tx) error {
					txCtx := NewContext(ctx, WithTransaction(tx))
					return dispatch(txCtx)
				}); derr != nil {
					return derr
				}
			} else if derr := dispatch(ctx); derr != nil {
				return derr
			}
		}

		return err
	}
}
