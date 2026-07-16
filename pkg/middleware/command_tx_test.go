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
	ID cedar.EntityUID
}

func (e testEntity) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:        e.ID,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
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
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !sawTx {
		t.Fatalf("expected loader to run within a transaction")
	}
}
