package tui

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestInventoryColumnsExposeTagsOnlyWhenThereIsRoom(t *testing.T) {
	t.Parallel()
	narrow := inventoryColumns(71)
	wide := inventoryColumns(90)
	testutil.Equals(t, len(narrow), 5)
	testutil.Equals(t, len(wide), 6)
	testutil.Equals(t, wide[len(wide)-1].Title, "Tags")
	testutil.ErrorIf(t, narrow[0].Width < 16 || wide[0].Width < 16, "ingredient widths narrow=%d wide=%d", narrow[0].Width, wide[0].Width)
}
