package telemetry

import (
	"strings"
	"sync"
	"time"
)

func Memory() *MemoryMetrics {
	return &MemoryMetrics{
		counters:   make(map[string]*memoryCounter),
		histograms: make(map[string]*memoryHistogram),
		gauges:     make(map[string]*memoryGauge),
		labelNames: make(map[string][]string),
	}
}

type MemoryMetrics struct {
	mu         sync.RWMutex
	counters   map[string]*memoryCounter
	histograms map[string]*memoryHistogram
	gauges     map[string]*memoryGauge

	labelNames map[string][]string
}

func (m *MemoryMetrics) Counter(name string, labels ...string) Counter {
	m.mu.Lock()
	defer m.mu.Unlock()

	if c, ok := m.counters[name]; ok {
		return c
	}
	m.labelNames[name] = append([]string(nil), labels...)
	c := &memoryCounter{values: make(map[string]float64)}
	m.counters[name] = c
	return c
}

func (m *MemoryMetrics) Histogram(name string, labels ...string) Histogram {
	m.mu.Lock()
	defer m.mu.Unlock()

	if h, ok := m.histograms[name]; ok {
		return h
	}
	m.labelNames[name] = append([]string(nil), labels...)
	h := &memoryHistogram{values: make(map[string][]float64)}
	m.histograms[name] = h
	return h
}

func (m *MemoryMetrics) Gauge(name string, labels ...string) Gauge {
	m.mu.Lock()
	defer m.mu.Unlock()

	if g, ok := m.gauges[name]; ok {
		return g
	}
	m.labelNames[name] = append([]string(nil), labels...)
	g := &memoryGauge{values: make(map[string]float64)}
	m.gauges[name] = g
	return g
}

func (m *MemoryMetrics) CounterValue(name string, labelValues ...string) float64 {
	m.mu.RLock()
	c := m.counters[name]
	m.mu.RUnlock()
	if c == nil {
		return 0
	}
	return c.get(labelValues...)
}

func (m *MemoryMetrics) HistogramCount(name string, labelValues ...string) int {
	m.mu.RLock()
	h := m.histograms[name]
	m.mu.RUnlock()
	if h == nil {
		return 0
	}
	return h.count(labelValues...)
}

type memoryCounter struct {
	mu     sync.RWMutex
	values map[string]float64
}

func (c *memoryCounter) Inc(labelValues ...string) { c.Add(1, labelValues...) }

func (c *memoryCounter) Add(value float64, labelValues ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[labelKey(labelValues)] += value
}

func (c *memoryCounter) get(labelValues ...string) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.values[labelKey(labelValues)]
}

type memoryHistogram struct {
	mu     sync.RWMutex
	values map[string][]float64
}

func (h *memoryHistogram) Observe(value float64, labelValues ...string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	key := labelKey(labelValues)
	h.values[key] = append(h.values[key], value)
}

func (h *memoryHistogram) ObserveDuration(start time.Time, labelValues ...string) {
	h.Observe(time.Since(start).Seconds(), labelValues...)
}

func (h *memoryHistogram) count(labelValues ...string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.values[labelKey(labelValues)])
}

type memoryGauge struct {
	mu     sync.RWMutex
	values map[string]float64
}

func (g *memoryGauge) Set(value float64, labelValues ...string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.values[labelKey(labelValues)] = value
}

func (g *memoryGauge) Inc(labelValues ...string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.values[labelKey(labelValues)]++
}

func (g *memoryGauge) Dec(labelValues ...string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.values[labelKey(labelValues)]--
}

func labelKey(labelValues []string) string {
	return strings.Join(labelValues, "|")
}
