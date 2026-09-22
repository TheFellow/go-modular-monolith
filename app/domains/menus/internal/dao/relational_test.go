package dao

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	testutil "github.com/TheFellow/go-modular-monolith/pkg/testutil/assert"
	"github.com/cedar-policy/cedar-go"
)

// Reopening after each write exercises the disk representation, including
// nested collections, optional values and replacement of owned child rows.
func TestRelationalMenuRowReopenAndUpdate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "domain.db")
	row := MenuRow{ID: "menu", Name: "Nested menu", Status: "draft", CreatedAt: time.Date(2026, 9, 1, 12, 30, 0, 123, time.UTC), Items: []MenuItemRow{{DrinkID: cedar.EntityUID{Type: "Drink", ID: "drink"}, DisplayName: optional.Some(""), Price: optional.Some(money.NewPriceFromCents(1234, currency.USD)), Featured: true, SortOrder: 3}, {DrinkID: cedar.EntityUID{Type: "Drink", ID: "other"}}}}
	s, err := store.Open(ctx, path)
	relationalNoError(t, err)
	s.Register(ctx, MenuRow{})
	relationalNoError(t, s.Write(ctx, func(tx *store.Tx) error { return tx.Insert(&row) }))
	relationalNoError(t, s.Close())
	for version := range 2 {
		s, err = store.Open(ctx, path)
		relationalNoError(t, err)
		s.Register(ctx, MenuRow{})
		relationalNoError(t, s.Read(ctx, func(tx *store.Tx) error {
			got := MenuRow{ID: row.ID}
			relationalNoError(t, tx.Get(&got))
			testutil.ErrorIf(t, !reflect.DeepEqual(row, got), "roundtrip mismatch: got %#v, want %#v", got, row)
			return nil
		}))
		if version == 0 {
			row.Items = row.Items[:1]
			row.Items[0].Price = optional.None[money.Price]()
			row.Items[0].DisplayName = optional.Some("Updated")
			relationalNoError(t, s.Write(ctx, func(tx *store.Tx) error { return tx.Update(&row) }))
		}
		relationalNoError(t, s.Close())
	}
}

func TestRelationalSchemaAndPaginationIndex(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "schema.db")
	s, err := store.Open(ctx, path)
	relationalNoError(t, err)
	t.Cleanup(func() { relationalNoError(t, s.Close()) })
	Register(ctx, s)
	db, err := sql.Open("sqlite", path)
	relationalNoError(t, err)
	t.Cleanup(func() { relationalNoError(t, db.Close()) })
	var schema string
	relationalNoError(t, db.QueryRowContext(ctx, "SELECT sql FROM sqlite_schema WHERE type='table' AND name=?", "menus").Scan(&schema))
	testutil.ErrorIf(t, strings.Contains(strings.ToLower(schema), "payload"), "document payload column: %s", schema)
	testutil.ErrorIf(t, strings.Contains(strings.ToLower(schema), "json"), "JSON schema: %s", schema)
	rows, err := db.QueryContext(ctx, "SELECT name, sql FROM sqlite_schema WHERE type='table' AND name LIKE ?", "menus_%")
	relationalNoError(t, err)
	children := 0
	for rows.Next() {
		var name, ddl string
		relationalNoError(t, rows.Scan(&name, &ddl))
		testutil.ErrorIf(t, !strings.Contains(ddl, "REFERENCES"), "missing foreign key in %s: %s", name, ddl)
		testutil.ErrorIf(t, !strings.Contains(ddl, "ON DELETE CASCADE"), "missing cascade in %s: %s", name, ddl)
		testutil.ErrorIf(t, strings.Contains(strings.ToLower(ddl), "json"), "JSON schema in %s: %s", name, ddl)
		children++
	}
	relationalNoError(t, rows.Err())
	relationalNoError(t, rows.Close())
	testutil.ErrorIf(t, children == 0, "nested data must live in owned relational tables")
	plans, err := db.QueryContext(ctx, "EXPLAIN QUERY PLAN SELECT id FROM menus WHERE status='draft' AND id<'z' ORDER BY id DESC")
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
	testutil.ErrorIf(t, !strings.Contains(detail, "USING COVERING INDEX"), "pagination must use covering index: %s", detail)
	testutil.ErrorIf(t, strings.Contains(detail, "TEMP B-TREE"), "index must support pagination order: %s", detail)
}

func relationalNoError(t *testing.T, err error) {
	t.Helper()
	testutil.ErrorIf(t, err != nil, "unexpected error: %v", err)
}
