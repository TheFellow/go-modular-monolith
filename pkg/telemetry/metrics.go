package telemetry

import "time"

type Metrics interface {
	Counter(name string, labels ...string) Counter
	Histogram(name string, labels ...string) Histogram
	Gauge(name string, labels ...string) Gauge
}

type Counter interface {
	Inc(labelValues ...string)
	Add(value float64, labelValues ...string)
}

type Histogram interface {
	Observe(value float64, labelValues ...string)
	ObserveDuration(start time.Time, labelValues ...string)
}

type Gauge interface {
	Set(value float64, labelValues ...string)
	Inc(labelValues ...string)
	Dec(labelValues ...string)
}
