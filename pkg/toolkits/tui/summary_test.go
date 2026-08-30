package tui

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestListSummaryOmitsEmptyMetadata(t *testing.T) {
	t.Parallel()
	testutil.Equals(t, ListSummary("active", "", TagSummary("featured,region=west")), "active • tags: featured,region=west")
	testutil.Equals(t, TagSummary("  "), "")
}
