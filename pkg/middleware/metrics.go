package middleware

import (
	"reflect"
	"strings"
	"time"

	cedar "github.com/cedar-policy/cedar-go"

	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
)

type MetricsCollector struct {
	commandTotal    telemetry.Counter
	commandDuration telemetry.Histogram
	commandErrors   telemetry.Counter

	queryTotal    telemetry.Counter
	queryDuration telemetry.Histogram
	queryErrors   telemetry.Counter

	eventsDispatched telemetry.Counter
	eventsDuration   telemetry.Histogram
	eventsErrors     telemetry.Counter
}

func NewMetricsCollector(m telemetry.Metrics) *MetricsCollector {
	if m == nil {
		m = telemetry.Nop()
	}
	return &MetricsCollector{
		commandTotal:    m.Counter(telemetry.MetricCommandTotal, telemetry.LabelAction, telemetry.LabelResult),
		commandDuration: m.Histogram(telemetry.MetricCommandDuration, telemetry.LabelAction),
		commandErrors:   m.Counter(telemetry.MetricCommandErrors, telemetry.LabelAction),

		queryTotal:    m.Counter(telemetry.MetricQueryTotal, telemetry.LabelAction, telemetry.LabelResult),
		queryDuration: m.Histogram(telemetry.MetricQueryDuration, telemetry.LabelAction),
		queryErrors:   m.Counter(telemetry.MetricQueryErrors, telemetry.LabelAction),

		eventsDispatched: m.Counter(telemetry.MetricEventsDispatched, telemetry.LabelEventType),
		eventsDuration:   m.Histogram(telemetry.MetricEventsDuration, telemetry.LabelEventType),
		eventsErrors:     m.Counter(telemetry.MetricEventsErrors, telemetry.LabelEventType),
	}
}

var nopMetricsCollector = NewMetricsCollector(telemetry.Nop())

func CommandMetrics() CommandMiddleware {
	return func(ctx *Context, action cedar.EntityUID, _ cedar.Entity, next CommandNext) error {
		mc, ok := MetricsCollectorFromContext(ctx.Context)
		if !ok || mc == nil {
			mc = nopMetricsCollector
		}

		actionLabel := actionLabel(action)
		start := time.Now()

		err := next(ctx)

		mc.commandDuration.ObserveDuration(start, actionLabel)
		if err != nil {
			mc.commandTotal.Inc(actionLabel, "error")
			mc.commandErrors.Inc(actionLabel)
		} else {
			mc.commandTotal.Inc(actionLabel, "success")
		}
		return err
	}
}

func QueryMetrics() QueryMiddleware {
	return func(ctx *Context, action cedar.EntityUID, next QueryNext) error {
		mc, ok := MetricsCollectorFromContext(ctx.Context)
		if !ok || mc == nil {
			mc = nopMetricsCollector
		}

		actionLabel := actionLabel(action)
		start := time.Now()

		err := next(ctx)

		mc.queryDuration.ObserveDuration(start, actionLabel)
		if err != nil {
			mc.queryTotal.Inc(actionLabel, "error")
			mc.queryErrors.Inc(actionLabel)
		} else {
			mc.queryTotal.Inc(actionLabel, "success")
		}
		return err
	}
}

func QueryWithResourceMetrics() QueryWithResourceMiddleware {
	return func(ctx *Context, action cedar.EntityUID, _ cedar.Entity, next QueryWithResourceNext) error {
		mc, ok := MetricsCollectorFromContext(ctx.Context)
		if !ok || mc == nil {
			mc = nopMetricsCollector
		}

		actionLabel := actionLabel(action)
		start := time.Now()

		err := next(ctx)

		mc.queryDuration.ObserveDuration(start, actionLabel)
		if err != nil {
			mc.queryTotal.Inc(actionLabel, "error")
			mc.queryErrors.Inc(actionLabel)
		} else {
			mc.queryTotal.Inc(actionLabel, "success")
		}
		return err
	}
}

func (mc *MetricsCollector) RecordEvent(event any, duration time.Duration, err error) {
	if mc == nil {
		return
	}
	et := eventTypeLabel(event)
	mc.eventsDispatched.Inc(et)
	mc.eventsDuration.Observe(duration.Seconds(), et)
	if err != nil {
		mc.eventsErrors.Inc(et)
	}
}

func actionLabel(action cedar.EntityUID) string {
	// Mixology::Drink::Action::"create" -> Drink.create
	s := action.String()
	parts := strings.Split(s, "::")
	if len(parts) < 4 {
		return s
	}
	domain := parts[1]
	id := strings.Trim(parts[len(parts)-1], `"`)
	return domain + "." + id
}

func eventTypeLabel(event any) string {
	t := reflect.TypeOf(event)
	if t == nil {
		return ""
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	pkg := t.PkgPath()
	if pkg != "" {
		if i := strings.LastIndex(pkg, "/"); i >= 0 && i < len(pkg)-1 {
			pkg = pkg[i+1:]
		}
	}
	if pkg == "" {
		return t.Name()
	}
	return pkg + "." + t.Name()
}
