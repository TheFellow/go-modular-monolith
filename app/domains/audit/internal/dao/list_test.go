package dao

import (
	"context"
	"iter"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
	"github.com/mjl-/bstore"
	"github.com/segmentio/ksuid"
)

type testContext struct{ context.Context }

func (testContext) Transaction() (*bstore.Tx, bool) { return nil, false }

func TestListCursorUsesSameSecondKSUIDOrderAcrossMutations(t *testing.T) {
	t.Parallel()

	ctx := testContext{telemetry.WithMetrics(context.Background(), telemetry.Memory())}
	s, err := store.Open(ctx, filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	Register(ctx, s)
	d := New(s)

	ids := sameSecondIDs(t, 6)
	insertRows(t, ctx, s, ids[:5]...)

	first := collectIDs(t, d.List(ctx, ListFilter{}), 2)
	if want := []string{ids[4], ids[3]}; !slices.Equal(first, want) {
		t.Fatalf("first page = %v, want %v", first, want)
	}
	cursor := first[len(first)-1]

	// This row sorts before the cursor and must not shift into later pages.
	insertRows(t, ctx, s, ids[5])
	if err := s.Write(ctx, func(tx *bstore.Tx) error {
		return tx.Delete(&AuditEntryRow{ID: cursor})
	}); err != nil {
		t.Fatalf("delete cursor: %v", err)
	}

	second := collectIDs(t, d.List(ctx, ListFilter{BeforeID: cursor}), 2)
	if want := []string{ids[2], ids[1]}; !slices.Equal(second, want) {
		t.Fatalf("second page = %v, want %v", second, want)
	}
	if slices.Contains(second, ids[5]) {
		t.Fatalf("newer concurrent row %q appeared after cursor %q", ids[5], cursor)
	}
}

func sameSecondIDs(t *testing.T, count int) []string {
	t.Helper()
	when := time.Unix(1_800_000_000, 0)
	ids := make([]string, count)
	for i := range count {
		payload := make([]byte, 16)
		payload[len(payload)-1] = byte(i + 1)
		id, err := ksuid.FromParts(when, payload)
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = "aud-" + id.String()
	}
	return ids
}

func insertRows(t *testing.T, ctx context.Context, s *store.Store, ids ...string) {
	t.Helper()
	if err := s.Write(ctx, func(tx *bstore.Tx) error {
		for _, id := range ids {
			if err := tx.Insert(&AuditEntryRow{ID: id}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("insert rows: %v", err)
	}
}

func collectIDs(t *testing.T, seq iter.Seq2[*models.AuditEntry, error], limit int) []string {
	t.Helper()
	ids := make([]string, 0, limit)
	for entry, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, entry.ID.String())
		if len(ids) == limit {
			break
		}
	}
	return ids
}
