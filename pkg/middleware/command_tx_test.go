package middleware_test

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	drinksauthz "github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

type testEntity struct {
	ID         cedar.EntityUID
	Attributes cedar.RecordMap
}

func (e testEntity) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:        e.ID,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(e.Attributes),
		Tags:       cedar.NewRecord(nil),
	}
}

type transactionProbe struct {
	ID   int
	Kind string
}

func newTransactionTestStore(t *testing.T) (context.Context, *store.Store) {
	t.Helper()

	ctx := authn.ToContext(context.Background(), authn.Owner())
	ctx = log.ToContext(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)))
	s, err := store.Open(ctx, filepath.Join(t.TempDir(), "middleware.test.db"))
	testutil.Ok(t, err)
	s.Register(ctx, transactionProbe{})
	t.Cleanup(func() { testutil.Ok(t, s.Close()) })
	return ctx, s
}

func insertTransactionProbe(ctx store.Context, kind string) error {
	return store.Write(ctx, func(tx *bstore.Tx) error {
		return tx.Insert(&transactionProbe{Kind: kind})
	})
}

func transactionProbeKinds(t *testing.T, ctx context.Context, s *store.Store) []string {
	t.Helper()

	var kinds []string
	err := s.Read(ctx, func(tx *bstore.Tx) error {
		rows, err := bstore.QueryTx[transactionProbe](tx).List()
		if err != nil {
			return err
		}
		for _, row := range rows {
			kinds = append(kinds, row.Kind)
		}
		return nil
	})
	testutil.Ok(t, err)
	return kinds
}

func TestRunCommand_ActivityRecorderFailureRollsBackBusinessWrite(t *testing.T) {
	t.Parallel()

	ctx, s := newTransactionTestStore(t)
	recordCalls := 0
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{
		Store: s,
		RecordActivity: func(ctx *middleware.Context, activity middlewareevents.Activity) error {
			recordCalls++
			testutil.IsTrue(t, activity.Success)
			if err := insertTransactionProbe(ctx, "success-audit"); err != nil {
				return err
			}
			return errors.Internalf("audit unavailable")
		},
	})
	resource := testEntity{ID: cedar.NewEntityUID(drinksauthz.DrinkType, cedar.String("atomic-success"))}

	_, err := middleware.RunCommand(pipeline, middleware.NewContext(ctx), middleware.CommandSpec[testEntity, testEntity]{
		Action: drinksauthz.ActionCreate,
		Load: func(*middleware.Context) (testEntity, error) {
			return resource, nil
		},
		Handle: func(ctx *middleware.Context, in testEntity) (testEntity, error) {
			return in, insertTransactionProbe(ctx, "business-write")
		},
	})

	testutil.ErrorIsInternal(t, err)
	testutil.ErrorContains(t, err, "record activity")
	testutil.Equals(t, recordCalls, 1)
	testutil.Equals(t, transactionProbeKinds(t, ctx, s), []string(nil))
}

func TestRunCommand_FailureRollsBackThenPersistsFailedActivity(t *testing.T) {
	t.Parallel()

	ctx, s := newTransactionTestStore(t)
	var recorded []middlewareevents.Activity
	recorderHadTransaction := false
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{
		Store: s,
		RecordActivity: func(ctx *middleware.Context, activity middlewareevents.Activity) error {
			recorded = append(recorded, activity)
			_, recorderHadTransaction = ctx.Transaction()
			return insertTransactionProbe(ctx, "failure-audit")
		},
	})
	resource := testEntity{ID: cedar.NewEntityUID(drinksauthz.DrinkType, cedar.String("atomic-failure"))}

	_, err := middleware.RunCommand(pipeline, middleware.NewContext(ctx), middleware.CommandSpec[testEntity, testEntity]{
		Action: drinksauthz.ActionCreate,
		Load: func(*middleware.Context) (testEntity, error) {
			return resource, nil
		},
		Handle: func(ctx *middleware.Context, in testEntity) (testEntity, error) {
			if err := insertTransactionProbe(ctx, "business-write"); err != nil {
				return testEntity{}, err
			}
			ctx.TouchEntity(in.ID)
			return testEntity{}, errors.FailedPreconditionf("handler rejected")
		},
	})

	testutil.ErrorIsFailedPrecondition(t, err)
	testutil.Equals(t, transactionProbeKinds(t, ctx, s), []string{"failure-audit"})
	testutil.IsTrue(t, recorderHadTransaction)
	testutil.Equals(t, len(recorded), 1)
	testutil.IsFalse(t, recorded[0].Success)
	testutil.Equals(t, recorded[0].Resource, resource.ID)
	testutil.Equals(t, recorded[0].Touches, []cedar.EntityUID{resource.ID})
	testutil.StringContains(t, recorded[0].Error, "handler rejected")
}

func TestRunCommand_UsesCallerTransactionForBusinessAndSuccessActivity(t *testing.T) {
	t.Parallel()

	ctx, s := newTransactionTestStore(t)
	tx, err := s.Begin(ctx, true)
	testutil.Ok(t, err)
	t.Cleanup(func() {
		if tx != nil {
			testutil.Ok(t, s.Rollback(tx))
		}
	})

	var recorderTx *bstore.Tx
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{
		Store: s,
		RecordActivity: func(ctx *middleware.Context, _ middlewareevents.Activity) error {
			recorderTx, _ = ctx.Transaction()
			return insertTransactionProbe(ctx, "success-audit")
		},
	})
	resource := testEntity{ID: cedar.NewEntityUID(drinksauthz.DrinkType, cedar.String("caller-transaction"))}

	_, err = middleware.RunCommand(
		pipeline,
		middleware.NewContext(ctx).WithTransaction(tx),
		middleware.CommandSpec[testEntity, testEntity]{
			Action: drinksauthz.ActionCreate,
			Load: func(ctx *middleware.Context) (testEntity, error) {
				got, _ := ctx.Transaction()
				testutil.IsTrue(t, got == tx)
				return resource, nil
			},
			Handle: func(ctx *middleware.Context, in testEntity) (testEntity, error) {
				return in, insertTransactionProbe(ctx, "business-write")
			},
		},
	)
	testutil.Ok(t, err)
	testutil.IsTrue(t, recorderTx == tx)
	rows, err := bstore.QueryTx[transactionProbe](tx).List()
	testutil.Ok(t, err)
	testutil.Equals(t, len(rows), 2)
	testutil.Equals(t, []string{rows[0].Kind, rows[1].Kind}, []string{"business-write", "success-audit"})

	testutil.Ok(t, s.Rollback(tx))
	tx = nil
}

func TestRunCommand_AuthorizesLoadedResourceBeforeHandle(t *testing.T) {
	t.Parallel()

	fix := testutil.NewFixture(t)
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{
		Store:          fix.Store,
		RecordActivity: func(*middleware.Context, middlewareevents.Activity) error { return nil },
	})

	loaded := false
	handled := false
	_, err := middleware.RunCommand(pipeline, fix.ActorContext("anonymous"), middleware.CommandSpec[testEntity, testEntity]{
		Action: drinksauthz.ActionCreate,
		Load: func(*middleware.Context) (testEntity, error) {
			loaded = true
			return testEntity{
				ID: cedar.NewEntityUID(cedar.EntityType("Mixology::Drink"), cedar.String("stub")),
			}, nil
		},
		Handle: func(_ *middleware.Context, in testEntity) (testEntity, error) {
			handled = true
			return in, nil
		},
	})
	testutil.ErrorIsPermission(t, err)
	testutil.IsTrue(t, loaded)
	testutil.IsFalse(t, handled)
}

func TestRunCommand_AuthorizesResultAfterHandle(t *testing.T) {
	t.Parallel()

	fix := testutil.NewFixture(t)
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{
		Store:          fix.Store,
		RecordActivity: func(*middleware.Context, middlewareevents.Activity) error { return nil },
	})

	wine := testEntity{
		ID: cedar.NewEntityUID(cedar.EntityType("Mixology::Drink"), cedar.String("stub")),
		Attributes: cedar.RecordMap{
			"Category": cedar.String("wine"),
		},
	}
	handled := false
	_, err := middleware.RunCommand(pipeline, fix.ActorContext("sommelier"), middleware.CommandSpec[testEntity, testEntity]{
		Action: drinksauthz.ActionUpdate,
		Load: func(*middleware.Context) (testEntity, error) {
			return wine, nil
		},
		Handle: func(_ *middleware.Context, out testEntity) (testEntity, error) {
			handled = true
			out.Attributes["Category"] = cedar.String("beer")
			return out, nil
		},
	})
	testutil.ErrorIsPermission(t, err)
	testutil.IsTrue(t, handled)
}

func TestRunCommand_LoaderRunsInTransaction(t *testing.T) {
	t.Parallel()

	fix := testutil.NewFixture(t)
	ctx := fix.OwnerContext()
	_, ok := ctx.Transaction()
	testutil.IsFalse(t, ok)
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{
		Store:          fix.Store,
		RecordActivity: func(*middleware.Context, middlewareevents.Activity) error { return nil },
	})

	var gotTx *bstore.Tx
	_, err := middleware.RunCommand(pipeline, ctx, middleware.CommandSpec[testEntity, testEntity]{
		Action: drinksauthz.ActionCreate,
		Load: func(ctx *middleware.Context) (testEntity, error) {
			gotTx, _ = ctx.Transaction()
			return testEntity{
				ID: cedar.NewEntityUID(cedar.EntityType("Mixology::Drink"), cedar.String("stub")),
			}, nil
		},
		Handle: func(_ *middleware.Context, in testEntity) (testEntity, error) {
			return in, nil
		},
	})
	testutil.Ok(t, err)
	testutil.NotNil(t, gotTx)
}
