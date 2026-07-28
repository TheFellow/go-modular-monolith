package tui_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
)

func TestTagLabel(t *testing.T) {
	t.Parallel()
	testutil.Equals(t, tui.TagLabel(""), "(none)")
	testutil.Equals(t, tui.TagLabel("region=west,seasonal"), "region=west,seasonal")
}
