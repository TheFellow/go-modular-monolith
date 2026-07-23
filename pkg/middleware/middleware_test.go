package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"iter"
	"log/slog"
	"strings"
	"testing"

	drinksauthz "github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestPageQuery_FillsPagePastDeniedEntities(t *testing.T) {
	t.Parallel()

	fix := testutil.NewFixture(t)
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{Store: fix.Store})
	items := []testEntity{
		testDrink("beer-1", "beer"),
		testDrink("wine-1", "wine"),
		testDrink("beer-2", "beer"),
		testDrink("wine-2", "wine"),
		testDrink("wine-3", "wine"),
	}

	page, err := middleware.RunPageQuery(
		pipeline,
		fix.ActorContext("sommelier"),
		drinksauthz.ActionList,
		func(store.Context, struct{}, paging.Cursor) iter.Seq2[testEntity, error] {
			return func(yield func(testEntity, error) bool) {
				for _, item := range items {
					if !yield(item, nil) {
						return
					}
				}
			}
		},
		func(item testEntity) paging.Cursor { return paging.Cursor(item.ID.ID) },
		struct{}{},
		paging.Request{Limit: 2},
	)
	testutil.Ok(t, err)
	testutil.Equals(t, len(page.Items), 2)
	testutil.Equals(t, page.Items[0].ID.ID, cedar.String("wine-1"))
	testutil.Equals(t, page.Items[1].ID.ID, cedar.String("wine-2"))
	testutil.Equals(t, page.Next, paging.Cursor("wine-2"))
}

func testDrink(id, category string) testEntity {
	return testEntity{
		ID: cedar.NewEntityUID(drinksauthz.DrinkType, cedar.String(id)),
		Attributes: cedar.RecordMap{
			drinksauthz.DrinkCategoryAttr:    cedar.String(category),
			drinksauthz.DrinkDescriptionAttr: cedar.String(""),
			drinksauthz.DrinkGlassAttr:       cedar.String(""),
			drinksauthz.DrinkNameAttr:        cedar.String(id),
		},
	}
}

// testLogBuffer captures log output for assertions.
type testLogBuffer struct {
	buf bytes.Buffer
}

func (b *testLogBuffer) Write(p []byte) (int, error) {
	return b.buf.Write(p)
}

func (b *testLogBuffer) String() string {
	return b.buf.String()
}

func (b *testLogBuffer) Messages() []string {
	lines := strings.Split(strings.TrimSpace(b.buf.String()), "\n")
	var msgs []string
	for _, line := range lines {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err == nil {
			if msg, ok := m["msg"].(string); ok {
				msgs = append(msgs, msg)
			}
		}
	}
	return msgs
}

func (b *testLogBuffer) Contains(s string) bool {
	return strings.Contains(b.buf.String(), s)
}

func (b *testLogBuffer) Count(s string) int {
	return strings.Count(b.buf.String(), s)
}

func newTestContext(logBuf *testLogBuffer, mem *telemetry.MemoryMetrics) *middleware.Context {
	ctx := context.Background()

	// Add logger to context
	logger := slog.New(slog.NewJSONHandler(logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx = log.ToContext(ctx, logger)

	// Add telemetry to context
	ctx = telemetry.WithMetrics(ctx, mem)

	return middleware.NewContext(authn.ToContext(ctx, authn.Anonymous()))
}

type testEvent struct {
	Name string
}

// --- Request Logging Tests (Permission Error Handling) ---

func TestQueryLogging_PermissionError_LogsDenied(t *testing.T) {
	t.Parallel()

	logBuf := &testLogBuffer{}
	mem := telemetry.Memory()
	mctx := newTestContext(logBuf, mem)

	action := drinksauthz.ActionList

	chain := middleware.NewChain(
		middleware.Logging(),
	)

	err := chain.Execute(mctx, middleware.QueryOperation(action), func(_ *middleware.Context) error {
		return errors.Permissionf("access denied")
	})

	testutil.ErrorIsPermission(t, err)
	testutil.StringContains(t, logBuf.String(), "query denied")

	// But it should log the start
	testutil.StringContains(t, logBuf.String(), "query started")
}

func TestQueryLogging_OtherError_Logs(t *testing.T) {
	t.Parallel()

	logBuf := &testLogBuffer{}
	mem := telemetry.Memory()
	mctx := newTestContext(logBuf, mem)

	action := drinksauthz.ActionList

	chain := middleware.NewChain(
		middleware.Logging(),
	)

	err := chain.Execute(mctx, middleware.QueryOperation(action), func(_ *middleware.Context) error {
		return errors.Internalf("database error")
	})

	testutil.NotNil(t, err)

	// Non-permission errors SHOULD be logged
	testutil.StringContains(t, logBuf.String(), "query failed")
}

func TestCommandLogging_PermissionError_LogsDenied(t *testing.T) {
	t.Parallel()

	logBuf := &testLogBuffer{}
	mem := telemetry.Memory()
	mctx := newTestContext(logBuf, mem)

	action := drinksauthz.ActionCreate

	chain := middleware.NewChain(
		middleware.Logging(),
	)

	err := chain.Execute(mctx, middleware.CommandOperation(action), func(_ *middleware.Context) error {
		return errors.Permissionf("access denied")
	})

	testutil.ErrorIsPermission(t, err)
	testutil.StringContains(t, logBuf.String(), "command denied")
}

// --- Full Chain Integration Tests ---

func TestEntityQuery_AuthorizesLoadedResult(t *testing.T) {
	t.Parallel()

	logBuf := &testLogBuffer{}
	mem := telemetry.Memory()
	mctx := newTestContext(logBuf, mem)
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{Metrics: mem})

	executed := false
	_, err := middleware.RunEntityQuery(pipeline, mctx, drinksauthz.ActionUpdate, func(_ store.Context, _ struct{}) (testEntity, error) {
		executed = true
		return testEntity{
			ID: cedar.NewEntityUID(cedar.EntityType("Mixology::Drink"), cedar.String("stub")),
		}, nil
	}, struct{}{})
	testutil.ErrorIsPermission(t, err)
	testutil.IsTrue(t, executed)
}

func TestEntityQuery_ReturnsNotFoundWithoutAuthorization(t *testing.T) {
	t.Parallel()

	logBuf := &testLogBuffer{}
	mem := telemetry.Memory()
	mctx := newTestContext(logBuf, mem)
	pipeline := middleware.NewPipeline(middleware.PipelineConfig{Metrics: mem})

	_, err := middleware.RunEntityQuery(pipeline, mctx, drinksauthz.ActionUpdate, func(_ store.Context, _ struct{}) (testEntity, error) {
		return testEntity{}, errors.NotFoundf("drink missing")
	}, struct{}{})
	testutil.ErrorIsNotFound(t, err)
}

func TestTrackActivity_MissingCallbackFailsBeforeCommand(t *testing.T) {
	t.Parallel()

	logBuf := &testLogBuffer{}
	mem := telemetry.Memory()
	mctx := newTestContext(logBuf, mem)

	chain := middleware.NewChain(
		middleware.TrackActivity(nil, nil),
	)

	called := false
	err := chain.Execute(mctx, middleware.CommandOperation(drinksauthz.ActionCreate), func(_ *middleware.Context) error {
		called = true
		return nil
	})

	testutil.ErrorIsInternal(t, err)
	testutil.IsFalse(t, called)
}

// --- Test Event Dispatch Through Full Command Chain ---

type mockDispatcher struct {
	dispatched []any
}

func (m *mockDispatcher) Dispatch(_ *middleware.Context, event any) error {
	m.dispatched = append(m.dispatched, event)
	return nil
}

func TestDispatchEvents_DispatchesEvents(t *testing.T) {
	t.Parallel()

	logBuf := &testLogBuffer{}
	mem := telemetry.Memory()

	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx = log.ToContext(ctx, logger)
	ctx = telemetry.WithMetrics(ctx, mem)

	dispatcher := &mockDispatcher{}

	mctx := middleware.NewContext(authn.ToContext(ctx, authn.Anonymous()))

	action := drinksauthz.ActionCreate

	// Use just DispatchEvents middleware to test dispatch behavior
	chain := middleware.NewChain(
		middleware.DispatchEvents(dispatcher),
	)

	event := testEvent{Name: "created"}

	err := chain.Execute(mctx, middleware.CommandOperation(action), func(ctx *middleware.Context) error {
		ctx.AddEvent(event)
		return nil
	})

	testutil.Ok(t, err)
	testutil.Equals(t, len(dispatcher.dispatched), 1)
}

type cascadingDispatcher struct {
	dispatched []any
}

func (d *cascadingDispatcher) Dispatch(ctx *middleware.Context, event any) error {
	d.dispatched = append(d.dispatched, event)
	if len(d.dispatched) == 1 {
		ctx.AddEvent(testEvent{Name: "cascaded"})
	}
	return nil
}

func TestDispatchEvents_DoesNotCascadeNewEvents(t *testing.T) {
	t.Parallel()

	logBuf := &testLogBuffer{}
	mem := telemetry.Memory()

	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx = log.ToContext(ctx, logger)
	ctx = telemetry.WithMetrics(ctx, mem)

	dispatcher := &cascadingDispatcher{}

	mctx := middleware.NewContext(authn.ToContext(ctx, authn.Anonymous()))

	action := drinksauthz.ActionCreate

	chain := middleware.NewChain(
		middleware.DispatchEvents(dispatcher),
	)

	err := chain.Execute(mctx, middleware.CommandOperation(action), func(ctx *middleware.Context) error {
		ctx.AddEvent(testEvent{Name: "created"})
		return nil
	})
	testutil.Ok(t, err)
	testutil.Equals(t, len(dispatcher.dispatched), 1)
}
