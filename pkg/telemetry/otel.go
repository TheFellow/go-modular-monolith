package telemetry

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// OTEL adapts an OpenTelemetry meter to the Metrics interface.
func OTEL(meter metric.Meter) Metrics {
	return &otelMetrics{
		meter:      meter,
		counters:   make(map[string]*otelCounter),
		histograms: make(map[string]*otelHistogram),
		gauges:     make(map[string]*otelGauge),
	}
}

type otelMetrics struct {
	meter metric.Meter

	mu         sync.RWMutex
	counters   map[string]*otelCounter
	histograms map[string]*otelHistogram
	gauges     map[string]*otelGauge
}

func (m *otelMetrics) Counter(name string, labels ...string) Counter {
	key := instrumentKey(name, labels)

	m.mu.RLock()
	if c, ok := m.counters[key]; ok {
		m.mu.RUnlock()
		return c
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.counters[key]; ok {
		return c
	}

	inst, err := m.meter.Float64Counter(name)
	if err != nil {
		return nopCounter{}
	}
	c := &otelCounter{c: inst, labelKeys: append([]string(nil), labels...)}
	m.counters[key] = c
	return c
}

func (m *otelMetrics) Histogram(name string, labels ...string) Histogram {
	key := instrumentKey(name, labels)

	m.mu.RLock()
	if h, ok := m.histograms[key]; ok {
		m.mu.RUnlock()
		return h
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if h, ok := m.histograms[key]; ok {
		return h
	}

	inst, err := m.meter.Float64Histogram(name)
	if err != nil {
		return nopHistogram{}
	}
	h := &otelHistogram{h: inst, labelKeys: append([]string(nil), labels...)}
	m.histograms[key] = h
	return h
}

func (m *otelMetrics) Gauge(name string, labels ...string) Gauge {
	key := instrumentKey(name, labels)

	m.mu.RLock()
	if g, ok := m.gauges[key]; ok {
		m.mu.RUnlock()
		return g
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if g, ok := m.gauges[key]; ok {
		return g
	}

	inst, err := m.meter.Float64UpDownCounter(name)
	if err != nil {
		return nopGauge{}
	}
	g := &otelGauge{
		g:         inst,
		labelKeys: append([]string(nil), labels...),
		last:      make(map[string]float64),
	}
	m.gauges[key] = g
	return g
}

type otelCounter struct {
	c         metric.Float64Counter
	labelKeys []string
}

func (c *otelCounter) Inc(labelValues ...string) {
	c.Add(1, labelValues...)
}

func (c *otelCounter) Add(value float64, labelValues ...string) {
	c.c.Add(context.Background(), value, metric.WithAttributes(toAttrs(c.labelKeys, labelValues)...))
}

type otelHistogram struct {
	h         metric.Float64Histogram
	labelKeys []string
}

func (h *otelHistogram) Observe(value float64, labelValues ...string) {
	h.h.Record(context.Background(), value, metric.WithAttributes(toAttrs(h.labelKeys, labelValues)...))
}

func (h *otelHistogram) ObserveDuration(start time.Time, labelValues ...string) {
	h.Observe(time.Since(start).Seconds(), labelValues...)
}

type otelGauge struct {
	g         metric.Float64UpDownCounter
	labelKeys []string

	mu   sync.Mutex
	last map[string]float64
}

func (g *otelGauge) Set(value float64, labelValues ...string) {
	key := labelKey(labelValues)

	g.mu.Lock()
	prev := g.last[key]
	g.last[key] = value
	g.mu.Unlock()

	delta := value - prev
	if delta == 0 {
		return
	}
	g.g.Add(context.Background(), delta, metric.WithAttributes(toAttrs(g.labelKeys, labelValues)...))
}

func (g *otelGauge) Inc(labelValues ...string) {
	g.g.Add(context.Background(), 1, metric.WithAttributes(toAttrs(g.labelKeys, labelValues)...))
}

func (g *otelGauge) Dec(labelValues ...string) {
	g.g.Add(context.Background(), -1, metric.WithAttributes(toAttrs(g.labelKeys, labelValues)...))
}

func toAttrs(keys []string, values []string) []attribute.KeyValue {
	n := len(keys)
	if len(values) < n {
		n = len(values)
	}
	attrs := make([]attribute.KeyValue, 0, n)
	for i := 0; i < n; i++ {
		attrs = append(attrs, attribute.String(keys[i], values[i]))
	}
	return attrs
}

func instrumentKey(name string, labels []string) string {
	if len(labels) == 0 {
		return name
	}
	return name + "|" + labelKey(labels)
}
