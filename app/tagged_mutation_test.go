package app_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
)

type taggedIngredient struct {
	ID   entity.IngredientID
	Tags tag.Tags
}

func (i *taggedIngredient) EntityUID() cedar.EntityUID { return i.ID.EntityUID() }
func (i *taggedIngredient) SetTags(values tag.Tags)    { i.Tags = values }

func TestRunTaggedMutationCommitsDomainAndCompleteTagSetTogether(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	desired := tag.Tags{{Key: "region", Value: "west"}, {Key: "featured"}}

	result, err := app.RunTaggedMutation(f.App.App, f.OwnerContext(), &desired, func(ctx *middleware.Context) (*taggedIngredient, error) {
		created, err := f.App.Ingredients.Create(ctx, &models.Ingredient{Name: "Atomic", Category: models.CategorySpirit, Unit: "oz"})
		if err != nil {
			return nil, err
		}
		return &taggedIngredient{ID: created.ID}, nil
	})
	testutil.Ok(t, err)
	testutil.Equals(t, result.Tags, desired.Sorted())
	persisted, err := f.App.Ingredients.Get(f.OwnerContext(), result.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, persisted.Tags, desired.Sorted())
}

func TestRunTaggedMutationRejectsInvalidTagsBeforeMutation(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	invalid := tag.Tags{{Key: "region"}, {Key: "region", Value: "east"}}
	called := false

	_, err := app.RunTaggedMutation(f.App.App, f.OwnerContext(), &invalid, func(*middleware.Context) (*taggedIngredient, error) {
		called = true
		return &taggedIngredient{}, nil
	})
	testutil.ErrorIf(t, err == nil, "expected invalid tags")
	testutil.Equals(t, called, false)
}

func TestRunTaggedMutationRollsBackDomainMutationWhenTagReplacementFails(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	created, err := f.App.Ingredients.Create(f.OwnerContext(), &models.Ingredient{Name: "Before", Category: models.CategorySpirit, Unit: "oz"})
	testutil.Ok(t, err)
	auditBefore, err := f.App.Audit.Count(f.OwnerContext(), audit.ListRequest{})
	testutil.Ok(t, err)
	desired := tag.Tags{{Key: "region", Value: "east"}}

	_, err = app.RunTaggedMutation(f.App.App, f.OwnerContext(), &desired, func(ctx *middleware.Context) (*taggedIngredient, error) {
		_, updateErr := f.App.Ingredients.Update(ctx, &models.Ingredient{ID: created.ID, Name: "After", Category: models.CategorySpirit, Unit: "oz"})
		// A syntactically valid but nonexistent target forces replacement failure.
		return &taggedIngredient{ID: entity.NewIngredientID()}, updateErr
	})
	testutil.ErrorIf(t, err == nil, "expected tag replacement failure")
	persisted, err := f.App.Ingredients.Get(f.OwnerContext(), created.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, persisted.Name, "Before")
	auditAfter, err := f.App.Audit.Count(f.OwnerContext(), audit.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, auditAfter, auditBefore)
}

func TestRunTaggedMutationParticipatesInCallerTransaction(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	desired := tag.Tags{{Key: "region", Value: "west"}}
	tx, err := f.App.Store.Begin(f.OwnerContext(), true)
	testutil.Ok(t, err)
	rolledBack := false
	t.Cleanup(func() {
		if !rolledBack {
			_ = f.App.Store.Rollback(tx)
		}
	})
	txCtx := f.OwnerContext().WithTransaction(tx)

	result, err := app.RunTaggedMutation(f.App.App, txCtx, &desired, func(ctx *middleware.Context) (*taggedIngredient, error) {
		created, createErr := f.App.Ingredients.Create(ctx, &models.Ingredient{Name: "Outer transaction", Category: models.CategorySpirit, Unit: "oz"})
		if createErr != nil {
			return nil, createErr
		}
		return &taggedIngredient{ID: created.ID}, nil
	})
	testutil.Ok(t, err)
	testutil.Equals(t, result.Tags, desired)
	testutil.Ok(t, f.App.Store.Rollback(tx))
	rolledBack = true

	_, err = f.App.Ingredients.Get(f.OwnerContext(), result.ID)
	testutil.ErrorIf(t, err == nil, "caller rollback must remove the domain mutation and tags")
}
