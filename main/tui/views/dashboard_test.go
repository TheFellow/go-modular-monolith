package views

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestDashboardLayoutConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		width         int
		expectedWidth int
		expectedCols  int
	}{
		{name: "default width", width: 0, expectedWidth: 34, expectedCols: 2},
		{name: "very small", width: 3, expectedWidth: 0, expectedCols: 1},
		{name: "single column", width: 60, expectedWidth: 56, expectedCols: 1},
		{name: "two columns", width: 80, expectedWidth: 37, expectedCols: 2},
		{name: "wide", width: 120, expectedWidth: 57, expectedCols: 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := &Dashboard{width: tc.width}
			cardWidth, columns := d.layoutConfig()
			testutil.Equals(t, cardWidth, tc.expectedWidth)
			testutil.Equals(t, columns, tc.expectedCols)
		})
	}
}
