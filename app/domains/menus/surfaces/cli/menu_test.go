package cli

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestMenuItemJSONAndTextViewsPreservePrice(t *testing.T) {
	t.Parallel()
	item := models.MenuItem{
		DrinkID: entity.NewDrinkID(),
		Price:   optional.Some(money.NewPriceFromCents(1250, currency.USD)),
	}

	jsonView := FromDomainMenuItem(item)
	textView := ToMenuItemRows([]models.MenuItem{item})[0]
	testutil.ErrorIf(t, jsonView.Price != "$12.50", "JSON price = %q", jsonView.Price)
	testutil.ErrorIf(t, textView.Price != jsonView.Price, "text price %q differs from JSON price %q", textView.Price, jsonView.Price)
}
