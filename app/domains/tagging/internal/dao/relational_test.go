package dao

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/store"
	testutil "github.com/TheFellow/go-modular-monolith/pkg/testutil/assert"
)

func TestRelationalIdentityAndLookupIndexes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "schema.db")
	s, err := store.Open(ctx, path)
	relationalNoError(t, err)
	t.Cleanup(func() { relationalNoError(t, s.Close()) })
	RegisterSchema(ctx, s)
	first := entityTagRow{EntityType: "Drink", EntityID: "drink", Key: "season", Value: "summer"}
	duplicate := entityTagRow{EntityType: "Drink", EntityID: "drink", Key: "season", Value: "winter"}
	relationalNoError(t, s.Write(ctx, func(tx *store.Tx) error { return tx.Insert(&first) }))
	err = s.Write(ctx, func(tx *store.Tx) error { return tx.Insert(&duplicate) })
	testutil.ErrorIf(t, err == nil, "duplicate domain identity must violate the unique constraint")
	db, err := sql.Open("sqlite", path)
	relationalNoError(t, err)
	t.Cleanup(func() { relationalNoError(t, db.Close()) })
	for _, query := range []string{
		"SELECT entity_type,entity_id FROM entity_tags WHERE key='season' AND value='summer' ORDER BY entity_type,entity_id",
		"SELECT key FROM entity_tags WHERE entity_type='Drink' AND entity_id='drink' ORDER BY key",
	} {
		plans, err := db.QueryContext(ctx, "EXPLAIN QUERY PLAN "+query)
		relationalNoError(t, err)
		var detail string
		for plans.Next() {
			var id, parent, unused int
			var part string
			relationalNoError(t, plans.Scan(&id, &parent, &unused, &part))
			detail += part
		}
		relationalNoError(t, plans.Err())
		relationalNoError(t, plans.Close())
		testutil.ErrorIf(t, !strings.Contains(detail, "USING") || !strings.Contains(detail, "INDEX"), "lookup must use index: %s: %s", query, detail)
		testutil.ErrorIf(t, strings.Contains(detail, "TEMP B-TREE"), "index must support sort: %s: %s", query, detail)
	}
}
func relationalNoError(t *testing.T, err error) {
	t.Helper()
	testutil.ErrorIf(t, err != nil, "unexpected error: %v", err)
}
