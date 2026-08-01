package money_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestParsePriceUserFacingForms(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ input, want string }{
		{"$1.23", "$1.23"},
		{"$ 1.23", "$1.23"},
		{"USD 1.23", "$1.23"},
		{"1.23 usd", "$1.23"},
		{"EUR 1.23", "1.23 €"},
		{"1.23 EUR", "1.23 €"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got, err := money.ParsePrice(tc.input)
			testutil.ErrorIf(t, err != nil, "%v", err)
			testutil.ErrorIf(t, got.String() != tc.want, "ParsePrice(%q) = %q, want %q", tc.input, got.String(), tc.want)
		})
	}
}

func TestParsePriceRejectsMissingAndMalformedValues(t *testing.T) {
	t.Parallel()
	for _, input := range []string{"", "1.23", "US 1.23", "USD nope", "XYZ 1.23"} {
		{
			_, err := money.ParsePrice(input)
			testutil.ErrorIf(t, err == nil, "ParsePrice(%q) unexpectedly succeeded", input)
		}
	}
}
