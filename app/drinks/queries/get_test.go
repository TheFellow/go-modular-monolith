package queries_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/drinks/queries"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestGet_Found(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "drinks.json")

	const seed = `[
  { "id": "margarita", "name": "Margarita" },
  { "id": "old-fashioned", "name": "Old Fashioned", "deleted_at": "2025-01-01T00:00:00Z" }
]`

	err := os.WriteFile(path, []byte(seed), 0o644)
	testutil.ErrorIf(t, err != nil, "write seed: %v", err)

	q, err := queries.New(path)
	testutil.ErrorIf(t, err != nil, "new queries: %v", err)

	got, err := q.Get(context.Background(), "margarita")
	testutil.ErrorIf(t, err != nil, "get: %v", err)

	testutil.Equals(t, got, models.Drink{ID: "margarita", Name: "Margarita"})
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "drinks.json")

	const seed = `[
  { "id": "margarita", "name": "Margarita" }
]`

	err := os.WriteFile(path, []byte(seed), 0o644)
	testutil.ErrorIf(t, err != nil, "write seed: %v", err)

	q, err := queries.New(path)
	testutil.ErrorIf(t, err != nil, "new queries: %v", err)

	_, err = q.Get(context.Background(), "missing")
	testutil.ErrorIf(t, !errors.Is(err, queries.ErrNotFound), "expected ErrNotFound, got %v", err)
}

func TestGet_DeletedIsNotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "drinks.json")

	const seed = `[
  { "id": "old-fashioned", "name": "Old Fashioned", "deleted_at": "2025-01-01T00:00:00Z" }
]`

	err := os.WriteFile(path, []byte(seed), 0o644)
	testutil.ErrorIf(t, err != nil, "write seed: %v", err)

	q, err := queries.New(path)
	testutil.ErrorIf(t, err != nil, "new queries: %v", err)

	_, err = q.Get(context.Background(), "old-fashioned")
	testutil.ErrorIf(t, !errors.Is(err, queries.ErrNotFound), "expected ErrNotFound, got %v", err)
}
