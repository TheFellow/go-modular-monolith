package tui_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
)

func TestLabelOrUsesCallerFallbackForEmptyValue(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name, value, fallback, want string
	}{
		{name: "empty", fallback: "(none)", want: "(none)"},
		{name: "value", value: "region=west,seasonal", fallback: "(none)", want: "region=west,seasonal"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := tui.LabelOr(test.value, test.fallback)
			testutil.Equals(t, got, test.want)
		})
	}
}
