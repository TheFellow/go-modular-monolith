package app_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestDomainCommandCommitsTagsAndOneActivity(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	desired := tag.Tags{{Key: "region", Value: "west"}, {Key: "featured"}}
	before, err := f.Audit.Count(f.OwnerContext(), audit.ListRequest{})
	testutil.Ok(t, err)
	result, err := f.Ingredients.Create(f.OwnerContext(), &models.Ingredient{Name: "Atomic", Category: models.CategorySpirit, Unit: "oz"}, tag.Replace(&desired))
	testutil.Ok(t, err)
	testutil.Equals(t, result.Tags, desired.Sorted())
	persisted, err := f.Ingredients.Get(f.OwnerContext(), result.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, persisted.Tags, desired.Sorted())
	after, err := f.Audit.Count(f.OwnerContext(), audit.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, after, before+1)
	page, err := f.Audit.List(f.OwnerContext(), audit.ListRequest{})
	testutil.Ok(t, err)
	entry := page.Items[0]
	testutil.IsTrue(t, entry.Success)
	testutil.StringContains(t, entry.Action, "create")
	testutil.Equals(t, len(entry.Effects), 2)
}
func TestDomainCommandRejectsInvalidTags(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	invalid := tag.Tags{{Key: "region"}, {Key: "region", Value: "east"}}
	result, err := f.Ingredients.Create(f.OwnerContext(), &models.Ingredient{Name: "Invalid", Category: models.CategorySpirit, Unit: "oz"}, tag.Replace(&invalid))
	testutil.ErrorIsInvalid(t, err)
	testutil.IsTrue(t, result == nil)
}
func TestDomainCommandRollsBackOnStaleTagsAndRecordsOneFailure(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	created := testutil.CreateIngredient(t, f, models.Ingredient{Name: "Before", Category: models.CategorySpirit, Unit: "oz"})
	desired := tag.Tags{{Key: "region", Value: "east"}}
	before, err := f.Audit.Count(f.OwnerContext(), audit.ListRequest{})
	testutil.Ok(t, err)
	updated := *created
	updated.Name = "After"
	result, err := f.Ingredients.Update(f.OwnerContext(), &updated, tag.Replace(&desired, tag.Tags{{Key: "stale"}}))
	testutil.ErrorIsConflict(t, err)
	testutil.IsTrue(t, result == nil)
	persisted, err := f.Ingredients.Get(f.OwnerContext(), created.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, persisted, created)
	after, err := f.Audit.Count(f.OwnerContext(), audit.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, after, before+1)
}
func TestDomainCommandTagsParticipateInCallerRollback(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	tx, err := f.Store.Begin(f.OwnerContext(), true)
	testutil.Ok(t, err)
	defer func() { _ = f.Store.Rollback(tx) }()
	desired := tag.Tags{{Key: "region", Value: "west"}}
	result, err := f.Ingredients.Create(f.OwnerContext().WithTransaction(tx), &models.Ingredient{Name: "Rollback", Category: models.CategorySpirit, Unit: "oz"}, tag.Replace(&desired))
	testutil.Ok(t, err)
	testutil.Ok(t, f.Store.Rollback(tx))
	_, err = f.Ingredients.Get(f.OwnerContext(), result.ID)
	testutil.ErrorIsNotFound(t, err)
}

func TestCallerTransactionCannotComposeMultipleCommands(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	tx, err := f.Store.Begin(f.OwnerContext(), true)
	testutil.Ok(t, err)
	defer func() { _ = f.Store.Rollback(tx) }()
	ctx := f.OwnerContext().WithTransaction(tx)
	created, err := f.Ingredients.Create(ctx, &models.Ingredient{Name: "One command", Category: models.CategorySpirit, Unit: "oz"})
	testutil.Ok(t, err)
	result, err := f.App.Tags.Upsert(ctx, created.EntityUID(), tag.Tag{Key: "second-command"})
	testutil.ErrorIsFailedPrecondition(t, err)
	testutil.IsFalse(t, result.Changed)
	// A fresh middleware context around the same transaction cannot bypass ownership.
	_, err = f.Ingredients.Create(f.OwnerContext().WithTransaction(tx), &models.Ingredient{Name: "Second", Category: models.CategorySpirit, Unit: "oz"})
	testutil.ErrorIsFailedPrecondition(t, err)
	persisted, err := f.Ingredients.Get(ctx, created.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(persisted.Tags), 0)
}
