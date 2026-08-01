package tui_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
)

type listItem interface {
	Title() string
	Description() string
	FilterValue() string
}

func TestListItemPresentsAndPreservesTypedValue(t *testing.T) {
	t.Parallel()

	type record struct{ ID int }
	item := tui.NewListItem(record{ID: 42}, "Title", "Description", "search terms")

	var _ listItem = item
	if item.Value != (record{ID: 42}) {
		testutil.ErrorIf(t, true, "Value = %#v, want ID 42", item.Value)
	}
	if item.Title() != "Title" || item.Description() != "Description" || item.FilterValue() != "search terms" {
		testutil.ErrorIf(t, true, "presentation = %q, %q, %q", item.Title(), item.Description(), item.FilterValue())
	}
}

func TestListItemDefaultsFilterValueToTitle(t *testing.T) {
	t.Parallel()

	item := tui.NewListItem(struct{}{}, "Title", "Description", "")

	if item.FilterValue() != "Title" {
		testutil.ErrorIf(t, true, "FilterValue() = %q, want title", item.FilterValue())
	}
}
