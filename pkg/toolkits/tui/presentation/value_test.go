package presentation_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/presentation"
)

func TestLabelOrUsesFallbackForEmptyValue(t *testing.T) {
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
				got := presentation.LabelOr(test.value, "(none)")
				testutil.ErrorIf(t, got != test.want, "LabelOr(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}
