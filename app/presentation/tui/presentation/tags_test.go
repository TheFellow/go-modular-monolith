package presentation_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/presentation/tui/presentation"
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
			{
				got := presentation.TagLabel(test.value)
				testutil.ErrorIf(t, got != test.want, "TagLabel(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}
