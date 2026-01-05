# Sprint 019: Telemetry & Metrics

## Goal

Add telemetry infrastructure for metrics collection that integrates with the middleware architecture, enabling operational visibility into system behavior without coupling to specific backends.

## Problem

Without metrics:
1. No visibility into command/query throughput or latency
2. Can't identify slow operations or bottlenecks
3. No way to track authorization patterns (denied vs allowed)
4. No insight into event processing performance
5. Can't monitor system health in production

## Solution

A metrics abstraction in `pkg/telemetry` that:
1. Defines metric types (counters, histograms, gauges)
2. Integrates via middleware (domains remain metrics-free)
3. Supports multiple backends (in-memory, Prometheus, OpenTelemetry)
4. Has zero-allocation fast path when using nop collector
5. Provides built-in metrics for all middleware concerns

## Tasks

- [x] Create `pkg/telemetry/metrics.go` with Metrics interface
- [x] Create `pkg/telemetry` types (Counter/Histogram/Gauge)
- [x] Create `pkg/telemetry/nop.go` with no-op implementation
- [x] Create `pkg/telemetry/memory.go` with in-memory collector (for tests)
- [x] Create `pkg/telemetry/otel.go` + `pkg/telemetry/prometheus.go` (OpenTelemetry-first Prometheus exporter)
- [x] Add metrics middleware for commands and queries
- [x] Add metrics to event dispatcher
- [x] Add metrics to authorization
- [x] Wire metrics into App initialization
- [x] Add `/metrics` endpoint option for Prometheus scraping
- [x] Verify `go test ./...` passes

## Architecture

### Metrics Interface

```go
// pkg/telemetry/metrics.go
package telemetry

import "time"

// Metrics is the telemetry collection interface.
type Metrics interface {
    // Counter returns a counter metric.
    Counter(name string, labels ...string) Counter

    // Histogram returns a histogram metric for timing/distributions.
    Histogram(name string, labels ...string) Histogram

    // Gauge returns a gauge metric for point-in-time values.
    Gauge(name string, labels ...string) Gauge
}

// Counter counts occurrences.
type Counter interface {
    Inc(labelValues ...string)
    Add(value float64, labelValues ...string)
}

// Histogram records distributions (e.g., latencies).
type Histogram interface {
    Observe(value float64, labelValues ...string)
    ObserveDuration(start time.Time, labelValues ...string)
}

// Gauge records point-in-time values.
type Gauge interface {
    Set(value float64, labelValues ...string)
    Inc(labelValues ...string)
    Dec(labelValues ...string)
}
```

### Pre-defined Metric Names

```go
// pkg/telemetry/names.go
package telemetry

// Metric names - centralized for consistency.
const (
    // Command metrics
    MetricCommandTotal    = "mixology_command_total"
    MetricCommandDuration = "mixology_command_duration_seconds"
    MetricCommandErrors   = "mixology_command_errors_total"

    // Query metrics
    MetricQueryTotal    = "mixology_query_total"
    MetricQueryDuration = "mixology_query_duration_seconds"
    MetricQueryErrors   = "mixology_query_errors_total"

    // Authorization metrics
    MetricAuthZTotal   = "mixology_authz_decisions_total"
    MetricAuthZDenied  = "mixology_authz_denied_total"
    MetricAuthZLatency = "mixology_authz_duration_seconds"

    // Event metrics
    MetricEventsDispatched = "mixology_events_dispatched_total"
    MetricEventsDuration   = "mixology_events_duration_seconds"
    MetricEventsErrors     = "mixology_events_errors_total"

    // Store metrics
    MetricStoreReadDuration  = "mixology_store_read_duration_seconds"
    MetricStoreWriteDuration = "mixology_store_write_duration_seconds"
)

// Common labels
const (
    LabelDomain    = "domain"
    LabelAction    = "action"
    LabelEventType = "event_type"
    LabelResult    = "result"    // "success", "error"
    LabelDecision  = "decision"  // "allow", "deny"
)
```

### No-op Implementation

```go
// pkg/telemetry/nop.go
package telemetry

import "time"

// Nop returns a no-op metrics collector.
func Nop() Metrics {
    return nopMetrics{}
}

type nopMetrics struct{}

func (nopMetrics) Counter(string, ...string) Counter     { return nopCounter{} }
func (nopMetrics) Histogram(string, ...string) Histogram { return nopHistogram{} }
func (nopMetrics) Gauge(string, ...string) Gauge         { return nopGauge{} }

type nopCounter struct{}

func (nopCounter) Inc(...string)            {}
func (nopCounter) Add(float64, ...string)   {}

type nopHistogram struct{}

func (nopHistogram) Observe(float64, ...string)          {}
func (nopHistogram) ObserveDuration(time.Time, ...string) {}

type nopGauge struct{}

func (nopGauge) Set(float64, ...string) {}
func (nopGauge) Inc(...string)          {}
func (nopGauge) Dec(...string)          {}
```

### In-Memory Collector (Testing)

```go
// pkg/telemetry/memory.go
package telemetry

import (
    "sync"
    "time"
)

// Memory returns an in-memory metrics collector for testing.
func Memory() *MemoryMetrics {
    return &MemoryMetrics{
        counters:   make(map[string]*memoryCounter),
        histograms: make(map[string]*memoryHistogram),
        gauges:     make(map[string]*memoryGauge),
    }
}

type MemoryMetrics struct {
    mu         sync.RWMutex
    counters   map[string]*memoryCounter
    histograms map[string]*memoryHistogram
    gauges     map[string]*memoryGauge
}

func (m *MemoryMetrics) Counter(name string, labels ...string) Counter {
    m.mu.Lock()
    defer m.mu.Unlock()

    key := name
    if c, ok := m.counters[key]; ok {
        return c
    }
    c := &memoryCounter{values: make(map[string]float64)}
    m.counters[key] = c
    return c
}

// ... similar for Histogram and Gauge

// Inspection methods for tests
func (m *MemoryMetrics) CounterValue(name string, labelValues ...string) float64 {
    m.mu.RLock()
    defer m.mu.RUnlock()

    if c, ok := m.counters[name]; ok {
        return c.get(labelValues...)
    }
    return 0
}

func (m *MemoryMetrics) HistogramCount(name string, labelValues ...string) int {
    m.mu.RLock()
    defer m.mu.RUnlock()

    if h, ok := m.histograms[name]; ok {
        return h.count(labelValues...)
    }
    return 0
}

type memoryCounter struct {
    mu     sync.RWMutex
    values map[string]float64
}

func (c *memoryCounter) Inc(labelValues ...string) {
    c.Add(1, labelValues...)
}

func (c *memoryCounter) Add(value float64, labelValues ...string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    key := labelKey(labelValues)
    c.values[key] += value
}

func (c *memoryCounter) get(labelValues ...string) float64 {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.values[labelKey(labelValues)]
}

func labelKey(labelValues []string) string {
    // Create consistent key from label values
    return strings.Join(labelValues, "|")
}
```

### Prometheus Adapter

```go
// pkg/telemetry/prometheus.go
package telemetry

import (
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus returns a Prometheus-backed metrics collector.
func Prometheus(namespace string) Metrics {
    return &promMetrics{
        namespace: namespace,
        counters:  make(map[string]*prometheus.CounterVec),
        histograms: make(map[string]*prometheus.HistogramVec),
        gauges:    make(map[string]*prometheus.GaugeVec),
    }
}

type promMetrics struct {
    mu         sync.RWMutex
    namespace  string
    counters   map[string]*prometheus.CounterVec
    histograms map[string]*prometheus.HistogramVec
    gauges     map[string]*prometheus.GaugeVec
}

func (p *promMetrics) Counter(name string, labels ...string) Counter {
    p.mu.Lock()
    defer p.mu.Unlock()

    if c, ok := p.counters[name]; ok {
        return &promCounter{vec: c}
    }

    c := promauto.NewCounterVec(prometheus.CounterOpts{
        Namespace: p.namespace,
        Name:      name,
    }, labels)
    p.counters[name] = c
    return &promCounter{vec: c}
}

func (p *promMetrics) Histogram(name string, labels ...string) Histogram {
    p.mu.Lock()
    defer p.mu.Unlock()

    if h, ok := p.histograms[name]; ok {
        return &promHistogram{vec: h}
    }

    h := promauto.NewHistogramVec(prometheus.HistogramOpts{
        Namespace: p.namespace,
        Name:      name,
        Buckets:   prometheus.DefBuckets,
    }, labels)
    p.histograms[name] = h
    return &promHistogram{vec: h}
}

// ... Gauge implementation similar

type promCounter struct {
    vec *prometheus.CounterVec
}

func (c *promCounter) Inc(labelValues ...string) {
    c.vec.WithLabelValues(labelValues...).Inc()
}

func (c *promCounter) Add(value float64, labelValues ...string) {
    c.vec.WithLabelValues(labelValues...).Add(value)
}

type promHistogram struct {
    vec *prometheus.HistogramVec
}

func (h *promHistogram) Observe(value float64, labelValues ...string) {
    h.vec.WithLabelValues(labelValues...).Observe(value)
}

func (h *promHistogram) ObserveDuration(start time.Time, labelValues ...string) {
    h.vec.WithLabelValues(labelValues...).Observe(time.Since(start).Seconds())
}
```

### Metrics Middleware

```go
// pkg/middleware/metrics.go
package middleware

import (
    "strings"
    "time"

    cedar "github.com/cedar-policy/cedar-go"
    "github.com/TheFellow/go-modular-monolith/pkg/telemetry"
)

// MetricsCollector holds pre-registered metrics for middleware.
type MetricsCollector struct {
    commandTotal    telemetry.Counter
    commandDuration telemetry.Histogram
    commandErrors   telemetry.Counter

    queryTotal    telemetry.Counter
    queryDuration telemetry.Histogram
    queryErrors   telemetry.Counter
}

// NewMetricsCollector creates middleware metrics from a Metrics instance.
func NewMetricsCollector(m telemetry.Metrics) *MetricsCollector {
    return &MetricsCollector{
        commandTotal:    m.Counter(telemetry.MetricCommandTotal, telemetry.LabelAction, telemetry.LabelResult),
        commandDuration: m.Histogram(telemetry.MetricCommandDuration, telemetry.LabelAction),
        commandErrors:   m.Counter(telemetry.MetricCommandErrors, telemetry.LabelAction),

        queryTotal:    m.Counter(telemetry.MetricQueryTotal, telemetry.LabelAction, telemetry.LabelResult),
        queryDuration: m.Histogram(telemetry.MetricQueryDuration, telemetry.LabelAction),
        queryErrors:   m.Counter(telemetry.MetricQueryErrors, telemetry.LabelAction),
    }
}

// CommandMetrics records command execution metrics.
func CommandMetrics(mc *MetricsCollector) CommandMiddleware {
    return func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next CommandNext) error {
        actionName := extractActionName(action)
        start := time.Now()

        err := next(ctx)

        mc.commandDuration.ObserveDuration(start, actionName)

        if err != nil {
            mc.commandTotal.Inc(actionName, "error")
            mc.commandErrors.Inc(actionName)
        } else {
            mc.commandTotal.Inc(actionName, "success")
        }

        return err
    }
}

// QueryMetrics records query execution metrics.
func QueryMetrics(mc *MetricsCollector) QueryMiddleware {
    return func(ctx *Context, action cedar.EntityUID, next QueryNext) error {
        actionName := extractActionName(action)
        start := time.Now()

        err := next(ctx)

        mc.queryDuration.ObserveDuration(start, actionName)

        if err != nil {
            mc.queryTotal.Inc(actionName, "error")
            mc.queryErrors.Inc(actionName)
        } else {
            mc.queryTotal.Inc(actionName, "success")
        }

        return err
    }
}

func extractActionName(action cedar.EntityUID) string {
    // Mixology::Drink::Action::"create" -> "Drink.create"
    s := action.String()
    // Parse out domain and action name
    // Implementation details...
    return s
}
```

### Authorization Metrics

```go
// pkg/authz/metrics.go
package authz

import (
    "time"

    cedar "github.com/cedar-policy/cedar-go"
    "github.com/TheFellow/go-modular-monolith/pkg/telemetry"
)

type AuthZMetrics struct {
    total   telemetry.Counter
    denied  telemetry.Counter
    latency telemetry.Histogram
}

func NewAuthZMetrics(m telemetry.Metrics) *AuthZMetrics {
    return &AuthZMetrics{
        total:   m.Counter(telemetry.MetricAuthZTotal, telemetry.LabelAction, telemetry.LabelDecision),
        denied:  m.Counter(telemetry.MetricAuthZDenied, telemetry.LabelAction),
        latency: m.Histogram(telemetry.MetricAuthZLatency, telemetry.LabelAction),
    }
}

func (am *AuthZMetrics) Record(action cedar.EntityUID, allowed bool, duration time.Duration) {
    actionName := action.String()
    am.latency.Observe(duration.Seconds(), actionName)

    if allowed {
        am.total.Inc(actionName, "allow")
    } else {
        am.total.Inc(actionName, "deny")
        am.denied.Inc(actionName)
    }
}
```

### Event Dispatcher Metrics

```go
// pkg/dispatcher/metrics.go
package dispatcher

import (
    "reflect"
    "time"

    "github.com/TheFellow/go-modular-monolith/pkg/telemetry"
)

type DispatcherMetrics struct {
    dispatched telemetry.Counter
    duration   telemetry.Histogram
    errors     telemetry.Counter
}

func NewDispatcherMetrics(m telemetry.Metrics) *DispatcherMetrics {
    return &DispatcherMetrics{
        dispatched: m.Counter(telemetry.MetricEventsDispatched, telemetry.LabelEventType),
        duration:   m.Histogram(telemetry.MetricEventsDuration, telemetry.LabelEventType),
        errors:     m.Counter(telemetry.MetricEventsErrors, telemetry.LabelEventType),
    }
}

func (dm *DispatcherMetrics) Record(event any, duration time.Duration, err error) {
    eventType := reflect.TypeOf(event).String()
    dm.dispatched.Inc(eventType)
    dm.duration.Observe(duration.Seconds(), eventType)
    if err != nil {
        dm.errors.Inc(eventType)
    }
}
```

### App Integration

```go
// app/options.go (additions)

func WithMetrics(m telemetry.Metrics) Option {
    return func(a *App) {
        a.metrics = m
    }
}

// app/app.go
type App struct {
    // ... existing fields
    metrics telemetry.Metrics
}

func New(opts ...Option) *App {
    a := &App{
        metrics: telemetry.Nop(), // Default to no-op
    }
    for _, opt := range opts {
        opt(a)
    }

    // Create metrics collectors
    mc := middleware.NewMetricsCollector(a.metrics)
    authzMetrics := authz.NewAuthZMetrics(a.metrics)
    dispatchMetrics := dispatcher.NewDispatcherMetrics(a.metrics)

    // Wire into middleware chains...
    // ...
}
```

### CLI Integration

```go
// main/cli/cli.go (additions)

var metricsFlag = &cli.BoolFlag{
    Name:    "metrics",
    Value:   false,
    Usage:   "Enable Prometheus metrics endpoint on :9090/metrics",
    Sources: cli.EnvVars("MIXOLOGY_METRICS"),
}

// In Before hook when --metrics is set:
if cmd.Bool("metrics") {
    metrics := telemetry.Prometheus("mixology")
    // Start HTTP server for /metrics endpoint
    go func() {
        http.Handle("/metrics", promhttp.Handler())
        http.ListenAndServe(":9090", nil)
    }()
    opts = append(opts, app.WithMetrics(metrics))
}
```

## Example Metrics Output

```
# HELP mixology_command_total Total number of commands executed
# TYPE mixology_command_total counter
mixology_command_total{action="Drink.create",result="success"} 42
mixology_command_total{action="Drink.create",result="error"} 3
mixology_command_total{action="Order.place",result="success"} 127

# HELP mixology_command_duration_seconds Command execution duration
# TYPE mixology_command_duration_seconds histogram
mixology_command_duration_seconds_bucket{action="Drink.create",le="0.005"} 35
mixology_command_duration_seconds_bucket{action="Drink.create",le="0.01"} 40
mixology_command_duration_seconds_bucket{action="Drink.create",le="+Inf"} 45
mixology_command_duration_seconds_sum{action="Drink.create"} 0.234
mixology_command_duration_seconds_count{action="Drink.create"} 45

# HELP mixology_authz_denied_total Authorization denials
# TYPE mixology_authz_denied_total counter
mixology_authz_denied_total{action="Drink.create"} 7
mixology_authz_denied_total{action="Menu.publish"} 2

# HELP mixology_events_dispatched_total Events dispatched
# TYPE mixology_events_dispatched_total counter
mixology_events_dispatched_total{event_type="events.DrinkCreated"} 42
mixology_events_dispatched_total{event_type="events.OrderCompleted"} 127
```

## Test Utilities

```go
// pkg/testutil/metrics.go
package testutil

import "github.com/TheFellow/go-modular-monolith/pkg/telemetry"

// NewFixture additions
func NewFixture(t testing.TB) *Fixture {
    // ...
    metrics := telemetry.Memory()

    a := app.New(
        app.WithStore(s),
        app.WithMetrics(metrics),
    )

    return &Fixture{
        // ...
        Metrics: metrics,
    }
}

// Test example
func TestCommandMetrics(t *testing.T) {
    fix := testutil.NewFixture(t)

    _, err := fix.Drinks.Create(fix.Ctx, models.Drink{Name: "Margarita"})
    require.NoError(t, err)

    // Verify metrics recorded
    count := fix.Metrics.CounterValue(telemetry.MetricCommandTotal, "Drink.create", "success")
    assert.Equal(t, float64(1), count)
}
```

## Success Criteria

- `pkg/telemetry` package with Metrics interface
- No-op, in-memory, and Prometheus implementations
- Metrics middleware for commands and queries
- Authorization and event dispatch metrics
- Pre-defined metric names and labels
- CLI flag to enable Prometheus endpoint
- In-memory collector for test assertions
- No metrics code in domain packages
- `go test ./...` passes

## Dependencies

- Sprint 018 (Uniform logging)
