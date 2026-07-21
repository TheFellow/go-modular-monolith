package dao_test

import (
	"context"
	"iter"
	"path/filepath"
	"slices"
	"testing"
	"time"

	auditdao "github.com/TheFellow/go-modular-monolith/app/domains/audit/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/mjl-/bstore"
	"github.com/segmentio/ksuid"
)

type testContext struct {
	context.Context
	tx *bstore.Tx
}

func (c testContext) Transaction() (*bstore.Tx, bool) { return c.tx, c.tx != nil }

func TestListOrdersSameSecondKSUIDsByValue(t *testing.T) {
	t.Parallel()

	d, s, ctx := newDAO(t)
	ids := sameSecondIDs(t, 4)
	insertEntries(t, ctx, s, d, ids[1], ids[3], ids[0], ids[2])

	got := collectIDs(t, d.List(ctx, auditdao.ListFilter{}), len(ids))
	testutil.Equals(t, got, []string{ids[3], ids[2], ids[1], ids[0]})
}

func TestListCursorDoesNotRequireStoredCursorRow(t *testing.T) {
	t.Parallel()

	d, s, ctx := newDAO(t)
	ids := sameSecondIDs(t, 6)
	insertEntries(t, ctx, s, d, ids[0], ids[1], ids[2], ids[4])

	// The cursor value is intentionally absent, equivalent to its row having
	// been deleted. A concurrently inserted row that sorts before it must not
	// shift into the following page.
	cursor := ids[3]
	insertEntries(t, ctx, s, d, ids[5])
	got := collectIDs(t, d.List(ctx, auditdao.ListFilter{BeforeID: cursor}), 2)

	testutil.Equals(t, got, []string{ids[2], ids[1]})
	testutil.ErrorIf(t, slices.Contains(got, ids[5]), "newer row %q appeared after cursor %q", ids[5], cursor)
}

func newDAO(t *testing.T) (*auditdao.DAO, *store.Store, testContext) {
	t.Helper()
	ctx := testContext{Context: telemetry.WithMetrics(context.Background(), telemetry.Memory())}
	s, err := store.Open(ctx, filepath.Join(t.TempDir(), "audit.db"))
	testutil.Ok(t, err)
	t.Cleanup(func() { _ = s.Close() })
	auditdao.Register(ctx, s)
	return auditdao.New(s), s, ctx
}

func sameSecondIDs(t *testing.T, count int) []string {
	t.Helper()
	when := time.Unix(1_800_000_000, 0)
	ids := make([]string, count)
	for i := range count {
		payload := make([]byte, 16)
		payload[len(payload)-1] = byte(i + 1)
		id, err := ksuid.FromParts(when, payload)
		testutil.Ok(t, err)
		ids[i] = entity.PrefixAuditEntry + "-" + id.String()
	}
	return ids
}

func insertEntries(t *testing.T, ctx testContext, s *store.Store, d *auditdao.DAO, ids ...string) {
	t.Helper()
	err := s.Write(ctx, func(tx *bstore.Tx) error {
		txCtx := testContext{Context: ctx.Context, tx: tx}
		for _, rawID := range ids {
			id, err := entity.ParseAuditEntryID(rawID)
			if err != nil {
				return err
			}
			if err := d.Insert(txCtx, models.AuditEntry{ID: id}); err != nil {
				return err
			}
		}
		return nil
	})
	testutil.Ok(t, err)
}

func collectIDs(t *testing.T, seq iter.Seq2[*models.AuditEntry, error], limit int) []string {
	t.Helper()
	ids := make([]string, 0, limit)
	for entry, err := range seq {
		testutil.Ok(t, err)
		ids = append(ids, entry.ID.String())
		if len(ids) == limit {
			break
		}
	}
	return ids
}
