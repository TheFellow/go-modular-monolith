package currency_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestParseCurrency(t *testing.T) {
	t.Parallel()

	usd, err := currency.Parse("USD")
	testutil.Ok(t, err)
	testutil.Equals(t, usd.Code, "USD")

	_, err = currency.Parse("unknown")
	testutil.ErrorIsInvalid(t, err)
}

func TestCurrencyFormat(t *testing.T) {
	t.Parallel()

	testutil.Equals(t, currency.USD.Format("12.50"), "$12.50")
	testutil.Equals(t, currency.EUR.Format("12.50"), "12.50 €")
}

func TestCurrencyJSON(t *testing.T) {
	t.Parallel()
	t.Run("Known Codes round-trip", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			str  string
			want currency.Currency
		}{
			{"USD", currency.USD},
			{"EUR", currency.EUR},
		}
		for _, tc := range testCases {
			blob, err := json.Marshal(tc.str)
			testutil.Ok(t, err)
			testutil.Equals(t, string(blob), fmt.Sprintf("%q", tc.str))

			var got currency.Currency
			testutil.Ok(t, json.Unmarshal(blob, &got))
			testutil.Equals(t, got, tc.want)
		}
	})
	t.Run("Unknown Codes unmarshal error", func(t *testing.T) {
		t.Parallel()
		var got currency.Currency
		err := json.Unmarshal([]byte(`"unknown"`), &got)
		testutil.ErrorIsInvalid(t, err)
		testutil.ErrorIf(t, got != (currency.Currency{}), "got %v, want empty", got)
	})
}
