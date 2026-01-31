package telemetry

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// PrometheusConfig configures the built-in Prometheus exporter.
type PrometheusConfig struct {
	Addr string // default ":9090"
	Path string // default "/metrics"
}

type PrometheusServer struct {
	Metrics  Metrics
	Handler  http.Handler
	Shutdown func(context.Context) error
}

func NewPrometheus() (*PrometheusServer, error) {
	exp, err := prometheus.New()
	if err != nil {
		return nil, err
	}

	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exp))
	otel.SetMeterProvider(provider)

	return &PrometheusServer{
		Metrics:  OTEL(provider.Meter("mixology")),
		Handler:  promhttp.Handler(),
		Shutdown: provider.Shutdown,
	}, nil
}
