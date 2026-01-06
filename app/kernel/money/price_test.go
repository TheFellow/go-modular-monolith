package money_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/govalues/decimal"
)

func TestPriceValidate(t *testing.T) {
	t.Parallel()

	if err := (money.NewPriceFromCents(100, "USD")).Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if err := (money.Price{Amount: decimal.MustNew(-1, 0), Currency: "USD"}).Validate(); !errors.IsInvalid(err) {
		t.Fatalf("expected invalid, got %v", err)
	}
}

func TestPrice_Cents_RoundHalfUp(t *testing.T) {
	t.Parallel()

	p, err := money.NewPrice("1.025", "USD")
	if err != nil {
		t.Fatalf("NewPrice: %v", err)
	}
	cents, err := p.Cents()
	if err != nil {
		t.Fatalf("Cents: %v", err)
	}
	if cents != 103 {
		t.Fatalf("expected 103 cents, got %d", cents)
	}

	p, err = money.NewPrice("1.024", "USD")
	if err != nil {
		t.Fatalf("NewPrice: %v", err)
	}
	cents, err = p.Cents()
	if err != nil {
		t.Fatalf("Cents: %v", err)
	}
	if cents != 102 {
		t.Fatalf("expected 102 cents, got %d", cents)
	}
}

func TestPrice_String_RoundHalfUp(t *testing.T) {
	t.Parallel()

	p, err := money.NewPrice("1.025", "USD")
	if err != nil {
		t.Fatalf("NewPrice: %v", err)
	}
	if got := p.String(); got != "$1.03" {
		t.Fatalf("expected $1.03, got %q", got)
	}

	p, err = money.NewPrice("1.025", "eur")
	if err != nil {
		t.Fatalf("NewPrice: %v", err)
	}
	if got := p.String(); got != "EUR 1.03" {
		t.Fatalf("expected EUR 1.03, got %q", got)
	}
}

func TestPrice_Mul(t *testing.T) {
	t.Parallel()

	p := money.NewPriceFromCents(100, "USD")
	f, err := decimal.Parse("2.5")
	if err != nil {
		t.Fatalf("decimal.Parse: %v", err)
	}
	got, err := p.Mul(f)
	if err != nil {
		t.Fatalf("Mul: %v", err)
	}
	cents, err := got.Cents()
	if err != nil {
		t.Fatalf("Cents: %v", err)
	}
	if cents != 250 {
		t.Fatalf("expected 250 cents, got %d", cents)
	}
}

func TestPrice_SuggestedPrice_CeilToCent(t *testing.T) {
	t.Parallel()

	p := money.NewPriceFromCents(100, "USD")
	got, err := p.SuggestedPrice(0.70)
	if err != nil {
		t.Fatalf("SuggestedPrice: %v", err)
	}
	cents, err := got.Cents()
	if err != nil {
		t.Fatalf("Cents: %v", err)
	}
	if cents != 334 {
		t.Fatalf("expected 334 cents ($3.34), got %d", cents)
	}
}
