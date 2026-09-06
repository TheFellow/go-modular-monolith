package middleware

import (
	"context"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	events "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/segmentio/ksuid"
)

type workflowState struct {
	id         string
	activities []*events.Activity
}

// RunWorkflow owns the transaction and the post-rollback failure activity.
// Nested commands retain their own actions and share a correlation identity.
func RunWorkflow(ctx *Context, s *store.Store, name string, record func(*Context, events.Activity) error, run func(*Context) error) error {
	if tx, ok := ctx.Transaction(); ok && tx != nil {
		return run(ctx)
	}
	state := &workflowState{id: ksuid.New().String()}
	derived := *ctx
	derived.workflow = state
	err := s.Write(ctx, func(tx *store.Tx) error { return run(derived.WithTransaction(tx)) })
	if err == nil {
		return nil
	}
	failure := events.NewActivity(cedar.NewEntityUID("Mixology::Workflow::Action", cedar.String(name)), cedar.EntityUID{}, ctx.Principal())
	failure.WorkflowID = state.id
	for _, activity := range state.activities {
		for _, uid := range activity.Touches {
			failure.Touch(uid)
		}
		failure.Effects = append(failure.Effects, activity.Effects...)
		failure.Participants = append(failure.Participants, activity.Participants...)
	}
	failure.Complete(err)
	auditCtx := NewContext(context.WithoutCancel(ctx))
	auditCtx.principal = ctx.Principal()
	auditErr := s.Write(auditCtx, func(tx *store.Tx) error { return record(auditCtx.WithTransaction(tx), *failure) })
	return errors.Join(err, auditErr)
}
