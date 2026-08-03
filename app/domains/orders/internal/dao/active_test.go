package dao

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
	testutil "github.com/TheFellow/go-modular-monolith/pkg/testutil/assert"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

type activeIDsTestContext struct {
	context.Context
}

func (activeIDsTestContext) Transaction() (*bstore.Tx, bool) { return nil, false }

func TestActiveIDsFiltersAndDeduplicatesRequestedIDs(t *testing.T) {
	t.Parallel()

	ctx := activeIDsTestContext{telemetry.WithMetrics(context.Background(), telemetry.Memory())}
	s, err := store.Open(ctx, filepath.Join(t.TempDir(), "orders.db"))
	testutil.ErrorIf(t, err != nil, "open store: %v", err)
	t.Cleanup(func() {
		err := s.Close()
		testutil.ErrorIf(t, err != nil, "close store: %v", err)
	})
	Register(ctx, s)

	deletedAt := time.Now()
	err = s.Write(ctx, func(tx *bstore.Tx) error {
		return tx.Insert(
			&OrderRow{ID: "active"},
			&OrderRow{ID: "deleted", DeletedAt: &deletedAt},
		)
	})
	testutil.ErrorIf(t, err != nil, "insert rows: %v", err)

	requested := []cedar.String{"active", "deleted", "absent", "active"}
	values := activeIDValues(requested)
	testutil.ErrorIf(t, !slices.Equal(values, []string{"active", "deleted", "absent"}),
		"query ID values = %v, want unique values in request order", values)
	got, err := New(s, nil).ActiveIDs(ctx, requested)
	testutil.ErrorIf(t, err != nil, "active IDs: %v", err)
	testutil.ErrorIf(t, got.Len() != 1 || !got.Contains("active"), "active IDs = %v, want only active", got.Slice())
}
