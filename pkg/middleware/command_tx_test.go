package middleware_test

import (
	"testing"

	drinksauthz "github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
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
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{
		Store:          fix.Store,
		RecordActivity: func(*middleware.Context, middlewareevents.Activity) error { return nil },
	})

	var sawTx bool
	_, err := middleware.RunCommand(pipeline, ctx, middleware.CommandSpec[testEntity, testEntity]{
		Action: drinksauthz.ActionCreate,
		Load: func(ctx *middleware.Context) (testEntity, error) {
			_, sawTx = ctx.Transaction()
			return testEntity{
				ID: cedar.NewEntityUID(cedar.EntityType("Mixology::Drink"), cedar.String("stub")),
			}, nil
		},
		Handle: func(_ *middleware.Context, in testEntity) (testEntity, error) {
			return in, nil
		},
	})
	testutil.Ok(t, err)
	testutil.IsTrue(t, sawTx)
}
