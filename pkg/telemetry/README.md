# Metrics and telemetry

`pkg/telemetry` defines the small metrics contract used by middleware and persistence, along with
no-op, in-memory, OpenTelemetry, and Prometheus-backed implementations. It keeps application code
independent of a metrics SDK while letting each executable own server startup and shutdown.

This package records metrics only. It does not create traces, propagate spans into measurements,
or configure logging; the standard instrumentation path is described in the
[middleware guide](../middleware/README.md#default-pipelines).

## Data path

```text
executable -> Nop or NewPrometheus -> WithMetrics(process context)
                                     -> app.New / middleware / store
                                     -> FromContext -> Counter / Histogram / Gauge
                                     -> backend
```

`FromContext` always returns a usable `Metrics`: a missing context value or explicit nil backend
falls back to `Nop`. This lets libraries instrument work without branching on whether an
entrypoint enabled metrics.

## Metrics contract

Backends vend three positional-label instruments:

- `Counter`: `Inc` or `Add` a monotonically increasing value;
- `Histogram`: `Observe` a value or seconds elapsed since a start time; and
- `Gauge`: `Set`, `Inc`, or `Dec` a current value.

Declare label names when acquiring the instrument and pass values in the same order when recording:

```go
requests := metrics.Counter("mixology_widget_total", "operation", "result")
requests.Inc("create", "success")
```

The interface does not validate label count or cardinality. Keep labels bounded and never use
entity IDs, user input, or error messages as label values. Reuse the constants in `names.go` for
application-owned instruments so names and label keys stay stable.

## Backends

| Constructor       | Intended use                                 | Behavior                                                                                        |
| ----------------- | -------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| `Nop()`           | Metrics-disabled processes and safe defaults | Discards every observation.                                                                     |
| `Memory()`        | Deterministic tests                          | Thread-safe counters, histograms, and gauges with counter-value and histogram-count inspection. |
| `OTEL(meter)`     | An externally configured OpenTelemetry SDK   | Caches instruments by name and label declaration and records through the supplied meter.        |
| `NewPrometheus()` | Built-in local Prometheus endpoint           | Creates an OTEL Prometheus reader and returns metrics, an HTTP handler, and provider shutdown.  |

The OTEL gauge adapter uses an up/down counter and remembers the last value per label set so `Set`
can emit a delta. Instrument creation failures degrade that instrument to the no-op implementation.
Recordings currently use `context.Background`, so they are not associated with a caller's span.

`MemoryMetrics.CounterValue` and `HistogramCount` return zero for an unknown instrument or label
set. Its observations are safe across goroutines, making the backend suitable for application and
middleware tests.

## Prometheus lifecycle

`NewPrometheus` does not listen on a socket. The executable chooses the address and path, owns the
HTTP server, and shuts down both the server and meter provider. The constructor also installs its
provider as OpenTelemetry's global meter provider, so create at most one per process:

```go
prom, err := telemetry.NewPrometheus()
if err != nil {
	return err
}

mux := http.NewServeMux()
mux.Handle("/metrics", prom.Handler)
server := &http.Server{Addr: ":9090", Handler: mux}
go func() { _ = server.ListenAndServe() }()

ctx = telemetry.WithMetrics(ctx, prom.Metrics)
defer func() {
	_ = server.Shutdown(context.Background())
	_ = prom.Shutdown(context.Background())
}()
```

The CLI, TUI, and GUI all follow this ownership pattern. Only one process can bind the default
`:9090` address; runtime flags and environment behavior are listed in the
[feature guide](../../docs/features.md#runtime-configuration).

## Current instrumentation

The default command/query middleware emits totals, errors, and durations labeled by Cedar action;
totals also carry `result=success|error`. The store records managed read and write durations.
Action labels normalize a Cedar UID such as `Mixology::Drink::Action::"create"` to `Drink.create`.

`names.go` also reserves stable authorization and event metric names. Those constants are not
currently emitted by the pure authorization evaluator or generated dispatcher; do not assume a
declared constant appears at `/metrics` until instrumentation uses it.

## Adding instrumentation

1. Add a stable `mixology_...` name and any reusable label keys to `names.go`.
2. Acquire the instrument once when constructing a long-lived component when possible.
3. Record bounded label values and durations in seconds.
4. Cover the success and error paths with `Memory()` assertions.
5. If lifecycle changes, verify every executable shuts down its server and provider.

Focused checks:

```sh
go test ./pkg/middleware ./pkg/store ./pkg/testutil
go test ./main/cli ./main/tui
go test -tags ci ./main/gui
```
