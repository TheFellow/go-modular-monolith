package menus_test

import (
	"testing"
	"time"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestMenus_CreateGetUpdateItemsPublishDraftDelete(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()

	base := b.WithIngredient("Menu Base", measurement.UnitOz)
	b.WithInventory(base, 10)
	drink := b.WithDrink(drinksmodels.Drink{
		Name: "House Sour", Category: drinksmodels.DrinkCategorySour, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: base.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}},
			Steps:       []string{"Shake"},
		},
	})

	count, err := f.Menus.Count(ctx, menus.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, count, 0)

	created, err := f.Menus.Create(ctx, &models.Menu{Name: "Dinner", Description: "Evening menu"})
	testutil.Ok(t, err)
	testutil.IsFalse(t, created.ID.IsZero())
	testutil.Equals(t, created.Status, models.MenuStatusDraft)

	got, err := f.Menus.Get(ctx, created.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got, created, cmpopts.EquateEmpty())

	updated, err := f.Menus.Update(ctx, &models.Menu{
		ID: created.ID, Name: "Late Dinner", Description: "After-hours menu",
	})
	testutil.Ok(t, err)
	wantUpdated := *created
	wantUpdated.Name = "Late Dinner"
	wantUpdated.Description = "After-hours menu"
	testutil.Equals(t, updated, &wantUpdated, cmpopts.EquateEmpty())

	updated, err = f.Menus.AddDrink(ctx, &models.MenuPatch{MenuID: created.ID, DrinkID: drink.ID})
	testutil.Ok(t, err)
	wantUpdated.Items = []models.MenuItem{{
		DrinkID: drink.ID, DisplayName: optional.None[string](), Price: optional.None[models.Price](),
		Availability: models.AvailabilityAvailable,
	}}
	testutil.Equals(t, updated, &wantUpdated, cmpopts.EquateEmpty())

	updated, err = f.Menus.Publish(ctx, &models.Menu{ID: created.ID})
	testutil.Ok(t, err)
	wantPublished := wantUpdated
	wantPublished.Status = models.MenuStatusPublished
	wantPublished.PublishedAt = updated.PublishedAt
	testutil.Equals(t, updated, &wantPublished, cmpopts.EquateEmpty())
	got, err = f.Menus.Get(ctx, created.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got, updated, cmpopts.EquateEmpty())

	updated, err = f.Menus.Draft(ctx, &models.Menu{ID: created.ID})
	testutil.Ok(t, err)
	wantDraft := wantPublished
	wantDraft.Status = models.MenuStatusDraft
	wantDraft.PublishedAt = optional.None[time.Time]()
	testutil.Equals(t, updated, &wantDraft, cmpopts.EquateEmpty())
	got, err = f.Menus.Get(ctx, created.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got, updated, cmpopts.EquateEmpty())

	updated, err = f.Menus.RemoveDrink(ctx, &models.MenuPatch{MenuID: created.ID, DrinkID: drink.ID})
	testutil.Ok(t, err)
	wantDraft.Items = nil
	testutil.Equals(t, updated, &wantDraft, cmpopts.EquateEmpty())

	deleted, err := f.Menus.Delete(ctx, created.ID)
	testutil.Ok(t, err)
	wantDeleted := wantDraft
	wantDeleted.Status = models.MenuStatusArchived
	wantDeleted.DeletedAt = deleted.DeletedAt
	testutil.Equals(t, deleted, &wantDeleted, cmpopts.EquateEmpty())
	_, err = f.Menus.Get(ctx, created.ID)
	testutil.ErrorIsNotFound(t, err)
	count, err = f.Menus.Count(ctx, menus.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, count, 0)
}
