package tui

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestStockStatusDefaultThresholdBoundaries(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		value float64
		want  string
	}{{0, "OUT"}, {10, "LOW"}, {10.01, "OK"}} {
		{
			got := stockStatus(measurement.MustAmount(tc.value, measurement.UnitOz))
			testutil.ErrorIf(t, got != tc.want, "stockStatus(%v) = %q, want %q", tc.value, got, tc.want)
		}
	}
}
