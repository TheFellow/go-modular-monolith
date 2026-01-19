package currency_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func TestParseCurrency(t *testing.T) {
	t.Parallel()

	usd, err := currency.Parse("usd")
	if err != nil {
		t.Fatalf("expected USD, got %v", err)
	}
	if usd.Code() != "USD" {
		t.Fatalf("expected USD code, got %s", usd.Code())
	}

	_, err = currency.Parse("unknown")
	if !errors.IsInvalid(err) {
		t.Fatalf("expected invalid for unknown currency, got %v", err)
	}
}

func TestCurrencyFormat(t *testing.T) {
	t.Parallel()

	if got := currency.USD.Format("12.50"); got != "$12.50" {
		t.Fatalf("expected $12.50, got %q", got)
	}
	if got := currency.EUR.Format("12.50"); got != "12.50 €" {
		t.Fatalf("expected 12.50 €, got %q", got)
	}
}

func TestParseCode(t *testing.T) {
	t.Parallel()

	code, err := currency.ParseCode(" usd ")
	if err != nil {
		t.Fatalf("expected USD code, got %v", err)
	}
	if code.Code() != "USD" {
		t.Fatalf("expected USD code, got %s", code.Code())
	}
	if got := code.Format("12.50"); got != "$12.50" {
		t.Fatalf("expected $12.50, got %q", got)
	}

	_, err = currency.ParseCode("")
	if !errors.IsInvalid(err) {
		t.Fatalf("expected invalid for empty currency, got %v", err)
	}
}
