package presentation_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/main/tui/presentation"
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
				t.Fatalf("TagLabel(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}
