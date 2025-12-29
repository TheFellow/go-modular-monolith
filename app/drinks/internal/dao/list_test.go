package dao_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestFileDrinkDAO_LoadThenList(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "drinks.json")

	const seed = `[
  { "id": "margarita", "name": "Margarita" },
  { "id": "old-fashioned", "name": "Old Fashioned", "deleted_at": "2025-01-01T00:00:00Z" },
  { "id": "negroni", "name": "Negroni" }
]`

	err := os.WriteFile(path, []byte(seed), 0o644)
	testutil.ErrorIf(t, err != nil, "write seed: %v", err)

	d := dao.NewFileDrinkDAO(path)
	err = d.Load(context.Background())
	testutil.ErrorIf(t, err != nil, "load: %v", err)

	drinks, err := d.List(context.Background())
	testutil.ErrorIf(t, err != nil, "list: %v", err)

	want := []dao.Drink{
		{ID: "margarita", Name: "Margarita"},
		{ID: "negroni", Name: "Negroni"},
	}

	testutil.Equals(t, drinks, want, cmpopts.SortSlices(func(a, b dao.Drink) bool { return a.ID < b.ID }))
}
