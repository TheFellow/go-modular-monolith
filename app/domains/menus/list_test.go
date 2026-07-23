package menus_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestMenus_ListExpressionFilters(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()

	target := b.WithPublishedMenu(models.Menu{
		Name: "Summer Terrace", Description: "Seasonal patio menu",
	})
	b.WithMenuModel(models.Menu{Name: "Winter Cellar", Description: "Rich winter menu"})

	tests := map[string]string{
		"id":          fmt.Sprintf("id == %q", target.ID.String()),
		"name":        `name.contains("Summer")`,
		"description": `description.contains("patio")`,
		"status":      `status == "published"`,
		"created_at":  fmt.Sprintf("created_at == date(%q)", target.CreatedAt.Format(time.RFC3339Nano)),
	}
	for name, expression := range tests {
		ctx := f.ActorContext("owner")
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			page, err := f.Menus.List(ctx, menus.ListRequest{Filter: expression})
			testutil.Ok(t, err)
			testutil.Equals(t, len(page.Items), 1)
			testutil.Equals(t, page.Items[0].ID, target.ID)
		})
	}
}
