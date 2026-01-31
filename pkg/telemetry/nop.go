package telemetry

import "time"

func Nop() Metrics {
	return nopMetrics{}
}

type nopMetrics struct{}

func (nopMetrics) Counter(string, ...string) Counter     { return nopCounter{} }
func (nopMetrics) Histogram(string, ...string) Histogram { return nopHistogram{} }
func (nopMetrics) Gauge(string, ...string) Gauge         { return nopGauge{} }

type nopCounter struct{}

func (nopCounter) Inc(...string)          {}
func (nopCounter) Add(float64, ...string) {}

type nopHistogram struct{}

func (nopHistogram) Observe(float64, ...string) {}
func (nopHistogram) ObserveDuration(time.Time, ...string) {
}

type nopGauge struct{}

func (nopGauge) Set(float64, ...string) {}
func (nopGauge) Inc(...string)          {}
func (nopGauge) Dec(...string)          {}
