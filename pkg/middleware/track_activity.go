package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"

	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
)

func TrackActivity(s *store.Store, recordActivity func(*Context, middlewareevents.Activity) error) Middleware {
	return func(ctx *Context, op Operation, next Next) error {
		if op.Kind != OperationKindCommand {
			return next(ctx)
		}

		if recordActivity == nil {
			return errors.Internalf("record activity callback missing from pipeline")
		}

		activity := middlewareevents.NewActivity(op.Action, cedar.EntityUID{}, ctx.Principal())
		ctx.activity = activity

		err := next(ctx)
		// The command pipeline finalizes successful activities inside UnitOfWork.
		// A non-zero completion time therefore means recording was already
		// attempted and its result must be returned without a second attempt.
		if !activity.CompletedAt.IsZero() {
			return err
		}

		completeActivity(activity, err)

		record := func(recordCtx *Context) error {
			return recordCompletedActivity(recordCtx, recordActivity, *activity, err == nil)
		}

		if tx, ok := ctx.Transaction(); ok && tx != nil {
			// The caller owns an injected transaction and its rollback policy. Keep
			// the activity in that transaction rather than competing for a second
			// bbolt write transaction while the caller's transaction is still open.
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

// recordSuccessfulActivity runs inside UnitOfWork. Recording before the unit
// of work returns makes the command, its event handlers, and its success audit
// entry one atomic write. Failed commands bypass this middleware's recording;
// TrackActivity records their failure only after UnitOfWork has rolled back.
func recordSuccessfulActivity(recordActivity func(*Context, middlewareevents.Activity) error) Middleware {
	return func(ctx *Context, op Operation, next Next) error {
		if op.Kind != OperationKindCommand {
			return next(ctx)
		}

		if err := next(ctx); err != nil {
			return err
		}

		activity, ok := ctx.Activity()
		if !ok {
			return errors.Internalf("activity missing from command context")
		}
		completeActivity(activity, nil)
		return recordCompletedActivity(ctx, recordActivity, *activity, true)
	}
}

func completeActivity(activity *middlewareevents.Activity, err error) {
	if activity.Resource.IsZero() && len(activity.Touches) > 0 {
		activity.Resource = activity.Touches[0]
	}
	activity.Complete(err)
}

func recordCompletedActivity(
	ctx *Context,
	recordActivity func(*Context, middlewareevents.Activity) error,
	activity middlewareevents.Activity,
	commandSucceeded bool,
) error {
	if err := recordActivity(ctx, activity); err != nil {
		log.FromContext(ctx).Error("record activity", log.Err(err))
		if commandSucceeded {
			return errors.Internalf("record activity: %w", err)
		}
	}
	return nil
}
