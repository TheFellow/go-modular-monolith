package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	drinksauthz "github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
)

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

	// Create middleware context with metrics collector
	mc := middleware.NewMetricsCollector(mem)
	return middleware.NewContext(ctx,
		middleware.WithAnonymousPrincipal(),
		middleware.WithMetricsCollector(mc),
	)
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

	chain := middleware.NewQueryChain(
		middleware.QueryLogging(),
	)

	err := chain.Execute(mctx, action, func(_ *middleware.Context) error {
		return errors.Permissionf("access denied")
	})

	if !errors.IsPermission(err) {
		t.Fatalf("expected permission error, got %v", err)
	}

	if !logBuf.Contains("query denied") {
		t.Errorf("expected 'query denied' log, got: %s", logBuf.String())
	}

	// But it should log the start
	if !logBuf.Contains("query started") {
		t.Errorf("expected 'query started' log, got: %s", logBuf.String())
	}
}

func TestQueryLogging_OtherError_Logs(t *testing.T) {
	t.Parallel()

	logBuf := &testLogBuffer{}
	mem := telemetry.Memory()
	mctx := newTestContext(logBuf, mem)

	action := drinksauthz.ActionList

	chain := middleware.NewQueryChain(
		middleware.QueryLogging(),
	)

	err := chain.Execute(mctx, action, func(_ *middleware.Context) error {
		return errors.Internalf("database error")
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Non-permission errors SHOULD be logged
	if !logBuf.Contains("query failed") {
		t.Errorf("expected 'query failed' log for non-permission error, got: %s", logBuf.String())
	}
}

func TestCommandLogging_PermissionError_LogsDenied(t *testing.T) {
	t.Parallel()

	logBuf := &testLogBuffer{}
	mem := telemetry.Memory()
	mctx := newTestContext(logBuf, mem)

	action := drinksauthz.ActionCreate

	chain := middleware.NewCommandChain(
		middleware.CommandLogging(),
	)

	err := chain.Execute(mctx, action, func(_ *middleware.Context) error {
		return errors.Permissionf("access denied")
	})

	if !errors.IsPermission(err) {
		t.Fatalf("expected permission error, got %v", err)
	}

	if !logBuf.Contains("command denied") {
		t.Errorf("expected 'command denied' log, got: %s", logBuf.String())
	}
}

// --- Full Chain Integration Tests ---

func TestQueryChain_WithAuthZ_Denial_LogsOnceAndRecordsRequestMetrics(t *testing.T) {
	t.Parallel()

	logBuf := &testLogBuffer{}
	mem := telemetry.Memory()

	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx = log.ToContext(ctx, logger)
	ctx = telemetry.WithMetrics(ctx, mem)

	// Use anonymous principal which should be denied for create
	mctx := middleware.NewContext(ctx,
		middleware.WithPrincipal(authn.Anonymous()),
		middleware.WithMetricsCollector(middleware.NewMetricsCollector(mem)),
	)

	action := drinksauthz.ActionCreate

	// Use the full chain with AuthZ
	err := middleware.Query.Execute(mctx, action, func(_ *middleware.Context) error {
		t.Fatal("handler should not be called when authz denies")
		return nil
	})

	if !errors.IsPermission(err) {
		t.Fatalf("expected permission error, got %v", err)
	}

	denialCount := logBuf.Count("query denied")
	if denialCount != 1 {
		t.Errorf("expected exactly 1 'query denied' log, got %d", denialCount)
	}

	queryTotal := mem.CounterValue(telemetry.MetricQueryTotal, "Drink.create", "error")
	if queryTotal != 1 {
		t.Errorf("expected 1 query error metric, got %v", queryTotal)
	}

	queryDuration := mem.HistogramCount(telemetry.MetricQueryDuration, "Drink.create")
	if queryDuration != 1 {
		t.Errorf("expected 1 query duration observation, got %v", queryDuration)
	}
}

func TestQueryChain_WithAuthZ_AllowedRequest_MetricsRecorded(t *testing.T) {
	t.Parallel()

	logBuf := &testLogBuffer{}
	mem := telemetry.Memory()

	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx = log.ToContext(ctx, logger)
	ctx = telemetry.WithMetrics(ctx, mem)

	mctx := middleware.NewContext(ctx,
		middleware.WithPrincipal(authn.Anonymous()),
		middleware.WithMetricsCollector(middleware.NewMetricsCollector(mem)),
	)

	action := drinksauthz.ActionList
	handlerCalled := false

	err := middleware.Query.Execute(mctx, action, func(_ *middleware.Context) error {
		handlerCalled = true
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !handlerCalled {
		t.Error("expected handler to be called")
	}

	// Check query metrics
	queryTotal := mem.CounterValue(telemetry.MetricQueryTotal, "Drink.list", "success")
	if queryTotal != 1 {
		t.Errorf("expected 1 query success metric, got %v", queryTotal)
	}

	queryDuration := mem.HistogramCount(telemetry.MetricQueryDuration, "Drink.list")
	if queryDuration != 1 {
		t.Errorf("expected 1 query duration observation, got %v", queryDuration)
	}
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

	mctx := middleware.NewContext(ctx,
		middleware.WithAnonymousPrincipal(),
		middleware.WithMetricsCollector(middleware.NewMetricsCollector(mem)),
		middleware.WithEventDispatcher(dispatcher),
	)

	action := drinksauthz.ActionCreate

	// Use just DispatchEvents middleware to test dispatch behavior
	chain := middleware.NewCommandChain(
		middleware.DispatchEvents(),
	)

	event := testEvent{Name: "created"}

	err := chain.Execute(mctx, action, func(ctx *middleware.Context) error {
		ctx.AddEvent(event)
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(dispatcher.dispatched) != 1 {
		t.Errorf("expected 1 dispatched event, got %d", len(dispatcher.dispatched))
	}
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

	mctx := middleware.NewContext(ctx,
		middleware.WithAnonymousPrincipal(),
		middleware.WithMetricsCollector(middleware.NewMetricsCollector(mem)),
		middleware.WithEventDispatcher(dispatcher),
	)

	action := drinksauthz.ActionCreate

	chain := middleware.NewCommandChain(
		middleware.DispatchEvents(),
	)

	err := chain.Execute(mctx, action, func(ctx *middleware.Context) error {
		ctx.AddEvent(testEvent{Name: "created"})
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(dispatcher.dispatched) != 1 {
		t.Errorf("expected 1 dispatched event, got %d", len(dispatcher.dispatched))
	}
}
