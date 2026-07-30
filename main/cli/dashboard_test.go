//nolint:paralleltest // fresh-process integration tests deliberately serialize database lifecycles.
package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestStatusCommandFreshProcessTextJSONAndNonDisclosure(t *testing.T) {
	h := newCLIE2E(filepath.Join(t.TempDir(), "status.db"))
	fresh := h.Run("status", "--json")
	testutil.Ok(t, fresh.Err)
	var got app.DashboardAggregate
	testutil.Ok(t, json.Unmarshal([]byte(fresh.Stdout), &got))
	testutil.Equals(t, got.DrinkCount, 0)
	testutil.Equals(t, got.IngredientCount, 0)
	created := h.Run("ingredients", "create", "Secret Gin", "--category", "spirit", "--unit", "oz")
	testutil.Ok(t, created.Err)
	text := h.Run("status")
	testutil.Ok(t, text.Err)
	testutil.StringContains(t, text.Stdout, "DRINKS")
	testutil.StringContains(t, text.Stdout, "RECENT ACTIVITY")
	anonymous := h.As("anonymous").Run("status", "--json")
	testutil.Ok(t, anonymous.Err)
	var hidden app.DashboardAggregate
	testutil.Ok(t, json.Unmarshal([]byte(anonymous.Stdout), &hidden))
	testutil.Equals(t, hidden.IngredientCount, 1)
	testutil.ErrorIf(t, contains(anonymous.Stdout+anonymous.Stderr, "Secret Gin"), "anonymous dashboard disclosed entity data")
}
func contains(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
