package presentation_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/surfaces/tui/presentation"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestTagLabelUsesApplicationEmptyState(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name, value, want string
	}{
		{name: "empty", want: "(none)"},
		{name: "canonical tags", value: "region=west,seasonal", want: "region=west,seasonal"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := presentation.TagLabel(test.value); got != test.want {
				testutil.ErrorIf(t, true, "TagLabel(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}
