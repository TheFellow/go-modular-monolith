package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"

	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
)

func TrackActivity(s *store.Store, recorder ActivityRecorder) Middleware {
	return func(ctx *Context, op Operation, next Next) error {
		if op.Kind != OperationKindCommand {
			return next(ctx)
		}

		if recorder == nil {
			return errors.Internalf("activity recorder missing from context")
		}

		activity := middlewareevents.NewActivity(op.Action, op.Resource.UID, ctx.Principal())
		ctx.activity = activity

		err := next(ctx)

		if activity.Resource.IsZero() {
			if len(activity.Touches) > 0 {
				activity.Resource = activity.Touches[0]
			}
		}
		activity.Complete(err)

		record := func(recordCtx *Context) error {
			if rerr := recorder.RecordActivity(recordCtx, *activity); rerr != nil {
				log.FromContext(recordCtx).Error("record activity", log.Err(rerr))
				if err == nil {
					return errors.Internalf("record activity: %w", rerr)
				}
			}
			return nil
		}

		if tx, ok := ctx.Transaction(); ok && tx != nil {
			if rerr := record(ctx); rerr != nil {
				return rerr
			}
		} else if s != nil {
			if rerr := s.Write(ctx, func(tx *bstore.Tx) error {
				txCtx := ctx.WithTransaction(tx)
				return record(txCtx)
			}); rerr != nil {
				return rerr
			}
		} else if rerr := record(ctx); rerr != nil {
			return rerr
		}

		return err
	}
}
