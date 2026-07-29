package cli

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func TestMenuItemJSONAndTextViewsPreservePrice(t *testing.T) {
	item := models.MenuItem{
		DrinkID: entity.NewDrinkID(),
		Price:   optional.Some(money.NewPriceFromCents(1250, currency.USD)),
	}

	jsonView := FromDomainMenuItem(item)
	textView := ToMenuItemRows([]models.MenuItem{item})[0]
	if jsonView.Price != "$12.50" {
		t.Fatalf("JSON price = %q", jsonView.Price)
	}
	if textView.Price != jsonView.Price {
		t.Fatalf("text price %q differs from JSON price %q", textView.Price, jsonView.Price)
	}
}
