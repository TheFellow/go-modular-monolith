package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	testutil "github.com/TheFellow/go-modular-monolith/pkg/testutil/assert"
)

func TestChangeMonitorSignalsCommittedWritesAndIgnoresRollback(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "changes.db")
	reader, err := Open(ctx, path)
	testutil.ErrorIf(t, err != nil, "open reader: %v", err)
	defer func() { _ = reader.Close() }()
	writer, err := Open(ctx, path)
	testutil.ErrorIf(t, err != nil, "open writer: %v", err)
	defer func() { _ = writer.Close() }()

	monitor, err := reader.MonitorChanges(ctx, 10*time.Millisecond)
	testutil.ErrorIf(t, err != nil, "monitor changes: %v", err)
	defer monitor.Close()

	err = writer.Write(ctx, func(tx *Tx) error {
		return tx.Insert(&revisionedRecord{ID: 1, Name: "committed"})
	})
	testutil.ErrorIf(t, err != nil, "commit write: %v", err)
	committedSignal := false
	select {
	case <-monitor.Signals():
		committedSignal = true
	case <-time.After(2 * time.Second):
	}
	testutil.ErrorIf(t, !committedSignal, "committed write did not produce an invalidation")
	testutil.ErrorIf(t, monitor.Epoch() == 0, "monitor epoch did not advance")

	tx, err := writer.Begin(ctx, true)
	testutil.ErrorIf(t, err != nil, "begin rollback: %v", err)
	err = tx.Insert(&revisionedRecord{ID: 2, Name: "rolled back"})
	testutil.ErrorIf(t, err != nil, "insert rollback row: %v", err)
	testutil.ErrorIf(t, writer.Rollback(tx) != nil, "rollback failed")
	rollbackSignal := false
	select {
	case <-monitor.Signals():
		rollbackSignal = true
	case <-time.After(80 * time.Millisecond):
	}
	testutil.ErrorIf(t, rollbackSignal, "rolled-back write produced an invalidation")
}
