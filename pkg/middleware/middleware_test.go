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
	cedar "github.com/cedar-policy/cedar-go"
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

// --- AuthZ Chain Tests ---

func TestAuthZChain_AllowedDecision_LogsAndRecordsMetrics(t *testing.T) {
	t.Parallel()

	logBuf := &testLogBuffer{}
	mem := telemetry.Memory()
	mctx := newTestContext(logBuf, mem)

	action := drinksauthz.ActionList

	// Anonymous can list - should be allowed
	chain := middleware.NewAuthZChain(
		middleware.AuthZLogging(),
		middleware.AuthZMetrics(),
	)

	err := chain.Execute(mctx, action, func() error {
		return nil // Simulate allowed
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check logging
	if !logBuf.Contains("authorization allowed") {
		t.Errorf("expected 'authorization allowed' log, got: %s", logBuf.String())
	}
	if logBuf.Contains("authorization denied") {
		t.Errorf("unexpected 'authorization denied' log, got: %s", logBuf.String())
	}

	// Check metrics
	allowCount := mem.CounterValue(telemetry.MetricAuthZTotal, "Drink.list", "allow")
	if allowCount != 1 {
		t.Errorf("expected 1 allow decision, got %v", allowCount)
	}

	latencyCount := mem.HistogramCount(telemetry.MetricAuthZLatency, "Drink.list")
	if latencyCount != 1 {
		t.Errorf("expected 1 latency observation, got %v", latencyCount)
	}
}

func TestAuthZChain_DeniedDecision_LogsAndRecordsMetrics(t *testing.T) {
	t.Parallel()

	logBuf := &testLogBuffer{}
	mem := telemetry.Memory()
	mctx := newTestContext(logBuf, mem)

	action := drinksauthz.ActionCreate

	chain := middleware.NewAuthZChain(
		middleware.AuthZLogging(),
		middleware.AuthZMetrics(),
	)

	err := chain.Execute(mctx, action, func() error {
		return errors.Permissionf("denied") // Simulate denial
	})

	if !errors.IsPermission(err) {
		t.Fatalf("expected permission error, got %v", err)
	}

	// Check logging
	if !logBuf.Contains("authorization denied") {
		t.Errorf("expected 'authorization denied' log, got: %s", logBuf.String())
	}

	// Check metrics
	denyCount := mem.CounterValue(telemetry.MetricAuthZTotal, "Drink.create", "deny")
	if denyCount != 1 {
		t.Errorf("expected 1 deny decision, got %v", denyCount)
	}

	deniedCount := mem.CounterValue(telemetry.MetricAuthZDenied, "Drink.create")
	if deniedCount != 1 {
		t.Errorf("expected 1 denied counter, got %v", deniedCount)
	}
}

func TestAuthZChain_ErrorDecision_LogsAndRecordsMetrics(t *testing.T) {
	t.Parallel()

	logBuf := &testLogBuffer{}
	mem := telemetry.Memory()
	mctx := newTestContext(logBuf, mem)

	action := drinksauthz.ActionCreate

	chain := middleware.NewAuthZChain(
		middleware.AuthZLogging(),
		middleware.AuthZMetrics(),
	)

	err := chain.Execute(mctx, action, func() error {
		return errors.Internalf("something went wrong") // Simulate error
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Check logging
	if !logBuf.Contains("authorization error") {
		t.Errorf("expected 'authorization error' log, got: %s", logBuf.String())
	}

	// Check metrics
	errorCount := mem.CounterValue(telemetry.MetricAuthZTotal, "Drink.create", "error")
	if errorCount != 1 {
		t.Errorf("expected 1 error decision, got %v", errorCount)
	}
}

// --- Event Chain Tests ---

type testEvent struct {
	Name string
}

func TestEventChain_SuccessfulDispatch_RecordsMetrics(t *testing.T) {
	t.Parallel()

	logBuf := &testLogBuffer{}
	mem := telemetry.Memory()
	mctx := newTestContext(logBuf, mem)

	event := testEvent{Name: "test"}

	chain := middleware.NewEventChain(
		middleware.EventMetrics(),
	)

	err := chain.Execute(mctx, event, func() error {
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check metrics - event type includes package name
	dispatchedCount := mem.CounterValue(telemetry.MetricEventsDispatched, "middleware_test.testEvent")
	if dispatchedCount != 1 {
		t.Errorf("expected 1 dispatched event, got %v", dispatchedCount)
	}

	durationCount := mem.HistogramCount(telemetry.MetricEventsDuration, "middleware_test.testEvent")
	if durationCount != 1 {
		t.Errorf("expected 1 duration observation, got %v", durationCount)
	}
}

func TestEventChain_FailedDispatch_RecordsMetrics(t *testing.T) {
	t.Parallel()

	logBuf := &testLogBuffer{}
	mem := telemetry.Memory()
	mctx := newTestContext(logBuf, mem)

	event := testEvent{Name: "test"}

	chain := middleware.NewEventChain(
		middleware.EventMetrics(),
	)

	err := chain.Execute(mctx, event, func() error {
		return errors.Internalf("handler failed")
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Check metrics
	errorCount := mem.CounterValue(telemetry.MetricEventsErrors, "middleware_test.testEvent")
	if errorCount != 1 {
		t.Errorf("expected 1 event error, got %v", errorCount)
	}
}

// --- Request Logging Tests (Permission Error Handling) ---

func TestQueryLogging_PermissionError_SkipsLogging(t *testing.T) {
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

	// Request logging should NOT log permission errors
	// (AuthZ logging handles those)
	if logBuf.Contains("query failed") {
		t.Errorf("request logging should not log permission errors, got: %s", logBuf.String())
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

func TestCommandLogging_PermissionError_SkipsLogging(t *testing.T) {
	t.Parallel()

	logBuf := &testLogBuffer{}
	mem := telemetry.Memory()
	mctx := newTestContext(logBuf, mem)

	action := drinksauthz.ActionCreate
	resource := cedar.Entity{
		UID:        cedar.NewEntityUID(cedar.EntityType("Mixology::Drink::Catalog"), cedar.String("default")),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}

	chain := middleware.NewCommandChain(
		middleware.CommandLogging(),
	)

	err := chain.Execute(mctx, action, resource, func(_ *middleware.Context) error {
		return errors.Permissionf("access denied")
	})

	if !errors.IsPermission(err) {
		t.Fatalf("expected permission error, got %v", err)
	}

	// Request logging should NOT log permission errors
	if logBuf.Contains("command failed") {
		t.Errorf("request logging should not log permission errors, got: %s", logBuf.String())
	}
}

// --- Full Chain Integration Tests ---

func TestQueryChain_WithAuthZ_ExactlyOnceLoggingForDenial(t *testing.T) {
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

	// Check exactly-once logging for denial
	denialCount := logBuf.Count("authorization denied")
	if denialCount != 1 {
		t.Errorf("expected exactly 1 'authorization denied' log, got %d", denialCount)
	}

	// Query logging should NOT log the denial
	if logBuf.Contains("query failed") {
		t.Errorf("query logging should not log permission errors (handled by authz logging)")
	}

	// Check metrics are recorded
	denyMetric := mem.CounterValue(telemetry.MetricAuthZTotal, "Drink.create", "deny")
	if denyMetric != 1 {
		t.Errorf("expected 1 authz deny metric, got %v", denyMetric)
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

	// Check authz metrics
	authzAllow := mem.CounterValue(telemetry.MetricAuthZTotal, "Drink.list", "allow")
	if authzAllow != 1 {
		t.Errorf("expected 1 authz allow metric, got %v", authzAllow)
	}

	authzDuration := mem.HistogramCount(telemetry.MetricAuthZLatency, "Drink.list")
	if authzDuration != 1 {
		t.Errorf("expected 1 authz duration observation, got %v", authzDuration)
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

func TestDispatchEvents_RecordsMetrics(t *testing.T) {
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
	resource := cedar.Entity{
		UID:        cedar.NewEntityUID(cedar.EntityType("Mixology::Drink::Catalog"), cedar.String("default")),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}

	// Use just DispatchEvents middleware to test event chain
	chain := middleware.NewCommandChain(
		middleware.DispatchEvents(middleware.EventMetrics()),
	)

	event := testEvent{Name: "created"}

	err := chain.Execute(mctx, action, resource, func(ctx *middleware.Context) error {
		ctx.AddEvent(event)
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(dispatcher.dispatched) != 1 {
		t.Errorf("expected 1 dispatched event, got %d", len(dispatcher.dispatched))
	}

	// Check event metrics
	dispatchedCount := mem.CounterValue(telemetry.MetricEventsDispatched, "middleware_test.testEvent")
	if dispatchedCount != 1 {
		t.Errorf("expected 1 event dispatched metric, got %v", dispatchedCount)
	}

	durationCount := mem.HistogramCount(telemetry.MetricEventsDuration, "middleware_test.testEvent")
	if durationCount != 1 {
		t.Errorf("expected 1 event duration observation, got %v", durationCount)
	}
}
