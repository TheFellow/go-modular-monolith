package money_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func TestPriceValidate(t *testing.T) {
	t.Parallel()

	if err := (money.Price{Amount: 100, Currency: "USD"}).Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if err := (money.Price{Amount: -1, Currency: "USD"}).Validate(); !errors.IsInvalid(err) {
		t.Fatalf("expected invalid, got %v", err)
	}
}
