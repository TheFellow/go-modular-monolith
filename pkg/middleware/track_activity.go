package middleware

import (
	"context"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"

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
		if ctx.workflow != nil {
			activity.WorkflowID = ctx.workflow.id
			ctx.workflow.activities = append(ctx.workflow.activities, activity)
		}

		err := next(ctx)
		// The command pipeline finalizes successful activities inside UnitOfWork.
		// A successful completed activity needs no second record. A failure
		// after that attempt (including audit/commit failure) is recorded after rollback.
		if !activity.CompletedAt.IsZero() && err == nil {
			return err
		}

		completeActivity(activity, err)

		record := func(recordCtx *Context) error {
			return recordCompletedActivity(recordCtx, recordActivity, *activity)
		}

		if tx, ok := ctx.Transaction(); ok && tx != nil {
			// The caller owns an injected transaction and its rollback policy. Keep
			// the activity in that transaction rather than competing for a second
			// SQLite write transaction while the caller's transaction is still open.
			if rerr := record(ctx); rerr != nil {
				return errors.Join(err, rerr)
			}
		} else if s != nil {
			auditCtx := *ctx
			auditCtx.Context = context.WithoutCancel(ctx.Context)
			if rerr := s.Write(&auditCtx, func(tx *store.Tx) error {
				txCtx := auditCtx.WithTransaction(tx)
				return record(txCtx)
			}); rerr != nil {
				return errors.Join(err, rerr)
			}
		} else if rerr := record(ctx); rerr != nil {
			return errors.Join(err, rerr)
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
		return recordCompletedActivity(ctx, recordActivity, *activity)
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
) error {
	if err := recordActivity(ctx, activity); err != nil {
		log.FromContext(ctx).Error("record activity", log.Err(err))
		return errors.Internalf("record activity: %w", err)
	}
	return nil
}
