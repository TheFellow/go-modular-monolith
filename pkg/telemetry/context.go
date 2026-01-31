package telemetry

import "context"

type metricsKey struct{}

var nop = Nop()

func WithMetrics(ctx context.Context, m Metrics) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if m == nil {
		m = nop
	}
	return context.WithValue(ctx, metricsKey{}, m)
}

func FromContext(ctx context.Context) Metrics {
	if ctx != nil {
		if m, ok := ctx.Value(metricsKey{}).(Metrics); ok && m != nil {
			return m
		}
	}
	return nop
}
