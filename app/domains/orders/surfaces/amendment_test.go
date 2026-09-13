package surfaces_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	presentation "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestAmendmentFormRejectsInvalidInputAndRetainsIdentity(t *testing.T) {
	t.Parallel()
	ingredient, replacement, drink := entity.NewIngredientID(), entity.NewIngredientID(), entity.NewDrinkID()
	order := models.Order{ID: entity.NewOrderID(), Revision: 4, Plan: []models.ItemSnapshot{{DrinkID: drink, Steps: []string{"Shake | gently"}, Ingredients: []models.IngredientSelection{{IngredientID: ingredient}}}}}
	form := presentation.NewAmendmentForm(order)
	form.Reason = "Guest approved"
	form.Replacements[0].ReplacementID = replacement.String()
	request, err := form.Request()
	testutil.Ok(t, err)
	testutil.Equals(t, request.Revision, uint64(4))
	testutil.Equals(t, request.OrderID, order.ID)
	testutil.Equals(t, len(request.Preparation), 0)
	for _, ratio := range []string{"NaN", "+Inf", "0", "-1", "wrong"} {
		form.Replacements[0].Ratio = ratio
		_, err := form.Request()
		testutil.ErrorIsInvalid(t, err)
	}
}

func TestAmendmentFormOnlyChangesEditedPreparationFields(t *testing.T) {
	t.Parallel()
	ingredient, replacement := entity.NewIngredientID(), entity.NewIngredientID()
	order := models.Order{ID: entity.NewOrderID(), Revision: 2, Plan: []models.ItemSnapshot{{DrinkID: entity.NewDrinkID(), Steps: []string{"  Shake\ncarefully  ", "  Strain  "}, Garnish: "Peel", Ingredients: []models.IngredientSelection{{IngredientID: ingredient}}}}}
	form := presentation.NewAmendmentForm(order)
	form.Reason, form.Replacements[0].ReplacementID = "Approved", replacement.String()
	request, err := form.Request()
	testutil.Ok(t, err)
	testutil.Equals(t, len(request.Preparation), 0)
	form.Preparation[0].Garnish = ""
	request, err = form.Request()
	testutil.Ok(t, err)
	testutil.Equals(t, len(request.Preparation), 1)
	testutil.Equals(t, request.Preparation[0].Steps, []string(nil))
	garnish, ok := request.Preparation[0].Garnish.Unwrap()
	testutil.Equals(t, ok, true)
	testutil.Equals(t, garnish, "")
	form.Preparation[0].Garnish = "Peel"
	form.Preparation[0].Steps = "Stir gently\nStrain"
	request, err = form.Request()
	testutil.Ok(t, err)
	testutil.Equals(t, request.Preparation[0].Steps, []string{"Stir gently", "Strain"})
	testutil.Equals(t, request.Preparation[0].Garnish.IsNone(), true)
}
