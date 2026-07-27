package views

import (
	"fmt"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/charmbracelet/lipgloss"
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

func TestDashboardRecentActivityFitsAssignedHeight(t *testing.T) {
	t.Parallel()

	recent := make([]AuditSummary, 10)
	for i := range recent {
		recent[i] = AuditSummary{
			Timestamp: fmt.Sprintf("12:%02d", i),
			Actor:     "owner",
			Action:    fmt.Sprintf("activity-%02d", i),
		}
	}
	d := &Dashboard{
		width:  100,
		height: 21, // Minimum application height after title and status bars.
		data:   &DashboardData{RecentActivity: recent},
	}

	view := d.View()
	testutil.ErrorIf(t, lipgloss.Height(view) > d.height,
		"dashboard height %d exceeded assigned height %d:\n%s", lipgloss.Height(view), d.height, view)
	testutil.StringContains(t, view, "activity-00")
	testutil.ErrorIf(t, strings.Contains(view, "activity-09"),
		"expected recent activity to be truncated to available height:\n%s", view)
}
