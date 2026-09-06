package app_test

import (
	"github.com/TheFellow/go-modular-monolith/app"
	im "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"testing"
)

func TestComposedEditorRejectsStaleTagsAndRollsBackDomainUpdate(t *testing.T) {
	t.Parallel()
	f, i, _, _ := workflowFixture(t)
	ctx := f.OwnerContext()
	expected := tag.Tags{}
	_, err := f.App.Tags.Upsert(ctx, i.EntityUID(), tag.Tag{Key: "concurrent"})
	testutil.Ok(t, err)
	desired := tag.Tags{{Key: "stale"}}
	_, err = app.RunTaggedMutation(f.App.App, ctx, &desired, func(ctx *middleware.Context) (*im.Ingredient, error) {
		updated := *i
		updated.Name = "Must roll back"
		return f.Ingredients.Update(ctx, &updated)
	}, expected)
	testutil.ErrorIsConflict(t, err)
	current, err := f.Ingredients.Get(ctx, i.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, current.Name, i.Name)
}
