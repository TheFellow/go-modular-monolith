package cli_test

import (
	"strings"
	"testing"

	orderscli "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces/cli"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestDecodePlacePreservesItemAndOrderNotes(t *testing.T) {
	t.Parallel()
	menuID := entity.NewMenuID()
	drinkID := entity.NewDrinkID()
	doc := `{"menu_id":"` + menuID.String() + `","items":[{"drink_id":"` + drinkID.String() + `","quantity":2,"notes":"less ice\nlemon twist"}],"notes":"bar seat\nanniversary"}`

	order, err := orderscli.DecodePlace(strings.NewReader(doc))
	testutil.Ok(t, err)
	testutil.Equals(t, order.MenuID, menuID)
	testutil.Equals(t, order.Items[0].DrinkID, drinkID)
	testutil.Equals(t, order.Items[0].Quantity, 2)
	testutil.Equals(t, order.Items[0].Notes, "less ice\nlemon twist")
	testutil.Equals(t, order.Notes, "bar seat\nanniversary")
}
