package tui

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
)

func TestStockStatusDefaultThresholdBoundaries(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		value float64
		want  string
	}{{0, "OUT"}, {10, "LOW"}, {10.01, "OK"}} {
		if got := stockStatus(measurement.MustAmount(tc.value, measurement.UnitOz)); got != tc.want {
			t.Fatalf("stockStatus(%v) = %q, want %q", tc.value, got, tc.want)
		}
	}
}
