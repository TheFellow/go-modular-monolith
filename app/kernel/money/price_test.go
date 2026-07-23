package money_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/govalues/decimal"
)

func TestPriceValidate(t *testing.T) {
	t.Parallel()

	testutil.Ok(t, (money.NewPriceFromCents(100, currency.USD)).Validate())
	testutil.ErrorIsInvalid(t, (money.Price{Amount: decimal.MustNew(-1, 0), Currency: currency.USD}).Validate())
}

func TestPrice_Cents_RoundHalfUp(t *testing.T) {
	t.Parallel()

	p, err := money.NewPrice("1.025", currency.USD)
	testutil.Ok(t, err)
	cents, err := p.Cents()
	testutil.Ok(t, err)
	testutil.Equals(t, cents, 103)

	p, err = money.NewPrice("1.024", currency.USD)
	testutil.Ok(t, err)
	cents, err = p.Cents()
	testutil.Ok(t, err)
	testutil.Equals(t, cents, 102)
}

func TestPrice_String_RoundHalfUp(t *testing.T) {
	t.Parallel()

	p, err := money.NewPrice("1.025", currency.USD)
	testutil.Ok(t, err)
	testutil.Equals(t, p.String(), "$1.03")

	p, err = money.NewPrice("1.025", currency.EUR)
	testutil.Ok(t, err)
	testutil.Equals(t, p.String(), "1.03 €")
}

func TestPrice_Mul(t *testing.T) {
	t.Parallel()

	p := money.NewPriceFromCents(100, currency.USD)
	f, err := decimal.Parse("2.5")
	testutil.Ok(t, err)
	got, err := p.Mul(f)
	testutil.Ok(t, err)
	cents, err := got.Cents()
	testutil.Ok(t, err)
	testutil.Equals(t, cents, 250)
}

func TestPrice_SuggestedPrice_CeilToCent(t *testing.T) {
	t.Parallel()

	p := money.NewPriceFromCents(100, currency.USD)
	got, err := p.SuggestedPrice(0.70)
	testutil.Ok(t, err)
	cents, err := got.Cents()
	testutil.Ok(t, err)
	testutil.Equals(t, cents, 334)
}
