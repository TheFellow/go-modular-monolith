//nolint:paralleltest // terminal program lifecycles intentionally run serially.
package tui

import (
	"fmt"
	"strings"
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func TestPlaceVMFiltersCatalogAndCombinesDuplicateLines(t *testing.T) {
	t.Parallel()
	gin := placeDrink{id: entity.NewDrinkID(), name: "Gin, dry"}
	v := newPlaceVM(nil, 1)
	v.loading = false
	v.menus = []placeMenu{{id: entity.NewMenuID(), name: "Late Night", drinks: []placeDrink{gin}}}
	v.menuQuery.SetValue("night")
	v.filterMenus()
	testutil.Equals(t, len(v.visibleMenus), 1)
	v.chooseMenu()
	v.drinkQuery.SetValue("gin,")
	v.filterDrinks()
	testutil.Equals(t, len(v.visibleDrinks), 1)
	v.quantity.SetValue("2")
	v.itemNotes.SetValue("first, stirred")
	v.add()
	v.quantity.SetValue("3")
	v.itemNotes.SetValue("second, stirred")
	v.add()
	testutil.Equals(t, len(v.lines), 1)
	testutil.Equals(t, v.lines[0].quantity, 5)
	testutil.Equals(t, v.lines[0].notes, "second, stirred")
}

func TestPlaceVMLoadsEveryPublishedMenuPage(t *testing.T) {
	fix := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, fix, ingredientsmodels.Ingredient{Name: "Unavailable Base", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	unavailable := testutil.CreateDrink(t, fix, drinksmodels.Drink{Name: "Unavailable Drink", Category: drinksmodels.DrinkCategoryHighball, Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}}, Steps: []string{"Build"}}})
	const published = 101
	for i := range published {
		testutil.CreateMenu(t, fix, fmt.Sprintf("Menu %03d", i), testutil.WithDrink(unavailable), testutil.Published())
	}
	testutil.CreateMenu(t, fix, "Unpublished")
	testutil.CreateMenu(t, fix, "Unavailable Published", testutil.WithDrink(unavailable), testutil.Published())
	testutil.SetInventory(t, fix, inventorymodels.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(0, ingredient.Unit), CostPerUnit: money.NewPriceFromCents(0, currency.USD)})

	v := newPlaceVM(fix.App, 1)
	msg := v.loadCatalog()().(placeCatalogLoadedMsg)
	testutil.Ok(t, msg.err)
	testutil.Equals(t, len(msg.menus), published+1)
	testutil.Equals(t, msg.menus[0].name, "Menu 000")
	testutil.Equals(t, msg.menus[published-1].name, "Menu 100")
	testutil.Equals(t, msg.menus[published].name, "Unavailable Published")
	testutil.Equals(t, len(msg.menus[published].drinks), 0)
}

func TestPlaceVMOwnsEachSearchKeyExactlyOnceAndAcceptsMultilineNotes(t *testing.T) {
	t.Parallel()
	v := newPlaceVM(nil, 1)
	v.loading = false
	v.menus = []placeMenu{{id: entity.NewMenuID(), name: "Comma, Club"}}

	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	testutil.Equals(t, v.menuQuery.Value(), "C")
	testutil.Equals(t, len(v.visibleMenus), 1)

	v.menu = &placeMenu{id: entity.NewMenuID(), name: "Menu"}
	v.field = placeFieldItemNotes
	v.refocus()
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("first")})
	v.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("second")})
	testutil.ErrorIf(t, !strings.Contains(v.itemNotes.Value(), "first\nsecond"), "multiline item notes were not retained: %q", v.itemNotes.Value())

	v.field = placeFieldOrderNotes
	v.refocus()
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("order")})
	v.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("note")})
	testutil.ErrorIf(t, !strings.Contains(v.orderNotes.Value(), "order\nnote"), "multiline order notes were not retained: %q", v.orderNotes.Value())
}

func TestPlaceVMRejectsInvalidQuantityAndProtectsDirtyBack(t *testing.T) {
	t.Parallel()
	drink := placeDrink{id: entity.NewDrinkID(), name: "Fizz"}
	v := newPlaceVM(nil, 1)
	v.menu = &placeMenu{id: entity.NewMenuID(), name: "Menu"}
	v.visibleDrinks = []placeDrink{drink}
	v.quantity.SetValue("0")
	v.add()
	testutil.ErrorIf(t, v.err == nil, "expected invalid quantity")
	testutil.ErrorIf(t, v.mayClose(), "first back must protect a dirty form")
	testutil.ErrorIf(t, !v.mayClose(), "second back must discard")
}
