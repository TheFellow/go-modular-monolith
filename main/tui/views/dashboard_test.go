package views

import "testing"

func TestDashboardLayoutConfig(t *testing.T) {
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
			d := &Dashboard{width: tc.width}
			cardWidth, columns := d.layoutConfig()
			if cardWidth != tc.expectedWidth || columns != tc.expectedCols {
				t.Fatalf("layoutConfig() = (%d, %d), want (%d, %d)", cardWidth, columns, tc.expectedWidth, tc.expectedCols)
			}
		})
	}
}
