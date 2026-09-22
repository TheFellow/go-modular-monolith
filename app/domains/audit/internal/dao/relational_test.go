package dao

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	testutil "github.com/TheFellow/go-modular-monolith/pkg/testutil/assert"
	"github.com/cedar-policy/cedar-go"
)

// Reopening after each write exercises the disk representation, including
// nested collections, optional values and replacement of owned child rows.
func TestRelationalAuditEntryRowReopenAndUpdate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "domain.db")
	row := AuditEntryRow{ID: "audit", Action: "Order::Amend", PrincipalType: "User", PrincipalID: "bartender", Success: true, Effects: []middlewareevents.Effect{{Kind: "amended", Resource: cedar.EntityUID{Type: "Order", ID: "order"}, Changes: []middlewareevents.Change{{Field: "steps", Before: "mix", After: "shake"}}}}, Participants: []cedar.EntityUID{{Type: "User", ID: "bartender"}}, Touches: []cedar.EntityUID{{Type: "Drink", ID: "drink"}}}
	s, err := store.Open(ctx, path)
	relationalNoError(t, err)
	s.Register(ctx, AuditEntryRow{})
	relationalNoError(t, s.Write(ctx, func(tx *store.Tx) error { return tx.Insert(&row) }))
	relationalNoError(t, s.Close())
	for version := range 2 {
		s, err = store.Open(ctx, path)
		relationalNoError(t, err)
		s.Register(ctx, AuditEntryRow{})
		relationalNoError(t, s.Read(ctx, func(tx *store.Tx) error {
			got := AuditEntryRow{ID: row.ID}
			relationalNoError(t, tx.Get(&got))
			testutil.ErrorIf(t, !reflect.DeepEqual(row, got), "roundtrip mismatch: got %#v, want %#v", got, row)
			return nil
		}))
		if version == 0 {
			row.Effects[0].Changes = append(row.Effects[0].Changes, middlewareevents.Change{Field: "garnish", After: "lime"})
			row.Participants = nil
			row.Touches = []cedar.EntityUID{}
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
	relationalNoError(t, db.QueryRowContext(ctx, "SELECT sql FROM sqlite_schema WHERE type='table' AND name=?", "audit_entries").Scan(&schema))
	testutil.ErrorIf(t, strings.Contains(strings.ToLower(schema), "payload"), "document payload column: %s", schema)
	testutil.ErrorIf(t, strings.Contains(strings.ToLower(schema), "json"), "JSON schema: %s", schema)
	rows, err := db.QueryContext(ctx, "SELECT name, sql FROM sqlite_schema WHERE type='table' AND name LIKE ?", "audit_entries_%")
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
	plans, err := db.QueryContext(ctx, "EXPLAIN QUERY PLAN SELECT id FROM audit_entries WHERE principal_type='User' AND principal_id='bartender' AND id<'z' ORDER BY id DESC")
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
