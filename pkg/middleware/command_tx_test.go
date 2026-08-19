package middleware_test

import (
	"context"
	"io"
	"log/slog"
	"maps"
	"path/filepath"
	"testing"

	drinksauthz "github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	ingredientauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
)

type testEntity struct {
	ID         cedar.EntityUID
	Attributes cedar.RecordMap
	Tags       cedar.RecordMap
}

func (e testEntity) CedarEntity() cedar.Entity {
	attrs := maps.Clone(e.Attributes)
	if e.ID.Type == drinksauthz.DrinkType {
		if attrs == nil {
			attrs = cedar.RecordMap{}
		}
		for _, name := range []cedar.String{
			drinksauthz.DrinkCategoryAttr,
			drinksauthz.DrinkDescriptionAttr,
			drinksauthz.DrinkGlassAttr,
			drinksauthz.DrinkNameAttr,
		} {
			if _, ok := attrs[name]; !ok {
				attrs[name] = cedar.String("")
			}
		}
	}
	return cedar.Entity{
		UID:        e.ID,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(attrs),
		Tags:       cedar.NewRecord(e.Tags),
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
	return store.Write(ctx, func(tx *store.Tx) error {
		return tx.Insert(&transactionProbe{Kind: kind})
	})
}

func transactionProbeKinds(t *testing.T, ctx context.Context, s *store.Store) []string {
	t.Helper()

	var kinds []string
	err := s.Read(ctx, func(tx *store.Tx) error {
		rows, err := store.QueryTx[transactionProbe](tx).List()
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

func TestLoadCommand_ActivityRecorderFailureRollsBackBusinessWrite(t *testing.T) {
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

	_, err := pipeline.LoadCommand(middleware.NewContext(ctx), drinksauthz.ActionCreate,
		func(*middleware.Context) (testEntity, error) {
			return resource, nil
		},
		func(ctx *middleware.Context, in testEntity) (testEntity, error) {
			return in, insertTransactionProbe(ctx, "business-write")
		},
	)

	testutil.ErrorIsInternal(t, err)
	testutil.ErrorContains(t, err, "record activity")
	testutil.Equals(t, recordCalls, 1)
	testutil.Equals(t, transactionProbeKinds(t, ctx, s), []string(nil))
}

func TestLoadCommand_FailureRollsBackThenPersistsFailedActivity(t *testing.T) {
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

	_, err := pipeline.LoadCommand(middleware.NewContext(ctx), drinksauthz.ActionCreate,
		func(*middleware.Context) (testEntity, error) {
			return resource, nil
		},
		func(ctx *middleware.Context, in testEntity) (testEntity, error) {
			if err := insertTransactionProbe(ctx, "business-write"); err != nil {
				return testEntity{}, err
			}
			ctx.TouchEntity(in.ID)
			return testEntity{}, errors.FailedPreconditionf("handler rejected")
		},
	)

	testutil.ErrorIsFailedPrecondition(t, err)
	testutil.Equals(t, transactionProbeKinds(t, ctx, s), []string{"failure-audit"})
	testutil.IsTrue(t, recorderHadTransaction)
	testutil.Equals(t, len(recorded), 1)
	testutil.IsFalse(t, recorded[0].Success)
	testutil.Equals(t, recorded[0].Resource, resource.ID)
	testutil.Equals(t, recorded[0].Touches, []cedar.EntityUID{resource.ID})
	testutil.StringContains(t, recorded[0].Error, "handler rejected")
}

func TestLoadCommand_UsesCallerTransactionForBusinessAndSuccessActivity(t *testing.T) {
	t.Parallel()

	ctx, s := newTransactionTestStore(t)
	tx, err := s.Begin(ctx, true)
	testutil.Ok(t, err)
	t.Cleanup(func() {
		if tx != nil {
			testutil.Ok(t, s.Rollback(tx))
		}
	})

	var recorderTx *store.Tx
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{
		Store: s,
		RecordActivity: func(ctx *middleware.Context, _ middlewareevents.Activity) error {
			recorderTx, _ = ctx.Transaction()
			return insertTransactionProbe(ctx, "success-audit")
		},
	})
	resource := testEntity{ID: cedar.NewEntityUID(drinksauthz.DrinkType, cedar.String("caller-transaction"))}

	_, err = pipeline.LoadCommand(
		middleware.NewContext(ctx).WithTransaction(tx),
		drinksauthz.ActionCreate,
		func(ctx *middleware.Context) (testEntity, error) {
			got, _ := ctx.Transaction()
			testutil.IsTrue(t, got == tx)
			return resource, nil
		},
		func(ctx *middleware.Context, in testEntity) (testEntity, error) {
			return in, insertTransactionProbe(ctx, "business-write")
		},
	)
	testutil.Ok(t, err)
	testutil.IsTrue(t, recorderTx == tx)
	rows, err := store.QueryTx[transactionProbe](tx).List()
	testutil.Ok(t, err)
	testutil.Equals(t, len(rows), 2)
	testutil.Equals(t, []string{rows[0].Kind, rows[1].Kind}, []string{"business-write", "success-audit"})

	testutil.Ok(t, s.Rollback(tx))
	tx = nil
}

func TestLoadCommand_AuthorizesLoadedResourceBeforeHandle(t *testing.T) {
	t.Parallel()

	fix := testutil.NewFixture(t)
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{
		Store:          fix.Store,
		RecordActivity: func(*middleware.Context, middlewareevents.Activity) error { return nil },
	})

	loaded := false
	handled := false
	_, err := pipeline.LoadCommand(fix.ActorContext("anonymous"), drinksauthz.ActionCreate,
		func(*middleware.Context) (testEntity, error) {
			loaded = true
			return testEntity{
				ID: cedar.NewEntityUID(cedar.EntityType("Mixology::Drink"), cedar.String("stub")),
			}, nil
		},
		func(_ *middleware.Context, in testEntity) (testEntity, error) {
			handled = true
			return in, nil
		},
	)
	testutil.ErrorIsPermission(t, err)
	testutil.IsTrue(t, loaded)
	testutil.IsFalse(t, handled)
}

func TestLoadCommand_AuthorizesResultAfterHandle(t *testing.T) {
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
	_, err := pipeline.LoadCommand(fix.ActorContext("sommelier"), drinksauthz.ActionUpdate,
		func(*middleware.Context) (testEntity, error) {
			return wine, nil
		},
		func(_ *middleware.Context, out testEntity) (testEntity, error) {
			handled = true
			out.Attributes["Category"] = cedar.String("beer")
			return out, nil
		},
	)
	testutil.ErrorIsPermission(t, err)
	testutil.IsTrue(t, handled)
}

func TestLoadCommand_AuthorizationActionsRequireEveryActionBeforeHandle(t *testing.T) {
	t.Parallel()

	fix := testutil.NewFixture(t)
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{
		Store:          fix.Store,
		RecordActivity: func(*middleware.Context, middlewareevents.Activity) error { return nil },
	})

	handled := false
	_, err := pipeline.LoadCommandActions(fix.ActorContext("manager"), drinksauthz.ActionTag,
		func(*middleware.Context) (testEntity, error) {
			return testEntity{
				ID:         cedar.NewEntityUID(drinksauthz.DrinkType, "stub"),
				Attributes: cedar.RecordMap{"Category": cedar.String("wine")},
			}, nil
		},
		func(_ *middleware.Context, in testEntity) (testEntity, error) {
			handled = true
			return in, nil
		},
		func(testEntity) []cedar.EntityUID {
			return []cedar.EntityUID{drinksauthz.ActionTag, ingredientauthz.ActionTag}
		},
	)
	testutil.ErrorIsPermission(t, err)
	testutil.IsFalse(t, handled)
}

func TestLoadCommand_AuthorizationActionsRequireEveryActionAfterHandle(t *testing.T) {
	t.Parallel()

	fix := testutil.NewFixture(t)
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{
		Store:          fix.Store,
		RecordActivity: func(*middleware.Context, middlewareevents.Activity) error { return nil },
	})

	handled := false
	_, err := pipeline.LoadCommandActions(fix.ActorContext("sommelier"), drinksauthz.ActionTag,
		func(*middleware.Context) (testEntity, error) {
			return testEntity{
				ID:         cedar.NewEntityUID(drinksauthz.DrinkType, "stub"),
				Attributes: cedar.RecordMap{"Category": cedar.String("wine")},
				Tags:       cedar.RecordMap{"audience": cedar.String("sommelier")},
			}, nil
		},
		func(_ *middleware.Context, out testEntity) (testEntity, error) {
			handled = true
			out.Attributes["Category"] = cedar.String("beer")
			return out, nil
		},
		func(testEntity) []cedar.EntityUID {
			return []cedar.EntityUID{drinksauthz.ActionGet, drinksauthz.ActionUpdate}
		},
	)
	testutil.ErrorIsPermission(t, err)
	testutil.IsTrue(t, handled)
}

func TestLoadCommand_EmptyAuthorizationActionsFailClosed(t *testing.T) {
	t.Parallel()

	fix := testutil.NewFixture(t)
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{
		Store:          fix.Store,
		RecordActivity: func(*middleware.Context, middlewareevents.Activity) error { return nil },
	})

	handled := false
	_, err := pipeline.LoadCommandActions(fix.OwnerContext(), drinksauthz.ActionTag,
		func(*middleware.Context) (testEntity, error) {
			return testEntity{ID: cedar.NewEntityUID(drinksauthz.DrinkType, "stub")}, nil
		},
		func(_ *middleware.Context, in testEntity) (testEntity, error) {
			handled = true
			return in, nil
		},
		func(testEntity) []cedar.EntityUID { return nil },
	)
	testutil.ErrorIsInternal(t, err)
	testutil.IsFalse(t, handled)
}

func TestLoadCommand_LoaderRunsInTransaction(t *testing.T) {
	t.Parallel()

	fix := testutil.NewFixture(t)
	ctx := fix.OwnerContext()
	_, ok := ctx.Transaction()
	testutil.IsFalse(t, ok)
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{
		Store:          fix.Store,
		RecordActivity: func(*middleware.Context, middlewareevents.Activity) error { return nil },
	})

	var gotTx *store.Tx
	_, err := pipeline.LoadCommand(ctx, drinksauthz.ActionCreate,
		func(ctx *middleware.Context) (testEntity, error) {
			gotTx, _ = ctx.Transaction()
			return testEntity{
				ID: cedar.NewEntityUID(cedar.EntityType("Mixology::Drink"), cedar.String("stub")),
			}, nil
		},
		func(_ *middleware.Context, in testEntity) (testEntity, error) {
			return in, nil
		},
	)
	testutil.Ok(t, err)
	testutil.NotNil(t, gotTx)
}

func TestCommand_HandlerRunsInTransaction(t *testing.T) {
	t.Parallel()

	fix := testutil.NewFixture(t)
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{
		Store:          fix.Store,
		RecordActivity: func(*middleware.Context, middlewareevents.Activity) error { return nil },
	})
	resource := testEntity{
		ID: cedar.NewEntityUID(cedar.EntityType("Mixology::Drink"), cedar.String("stub")),
	}

	var gotTx *store.Tx
	_, err := pipeline.Command(fix.OwnerContext(), drinksauthz.ActionCreate, resource,
		func(ctx *middleware.Context, in testEntity) (testEntity, error) {
			gotTx, _ = ctx.Transaction()
			return in, nil
		},
	)
	testutil.Ok(t, err)
	testutil.NotNil(t, gotTx)
}
