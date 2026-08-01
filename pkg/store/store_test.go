package store

import (
	"context"
	"path/filepath"
	"testing"

	testutil "github.com/TheFellow/go-modular-monolith/pkg/testutil/assert"
	"github.com/mjl-/bstore"
)

type transactionLifecycleRecord struct {
	ID   int
	Name string
}

func TestCommitPersistsAndUnregistersCallerTransaction(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "store.db"))
	if err != nil {
		testutil.ErrorIf(t, true, "open store: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			testutil.ErrorIf(t, true, "close store: %v", err)
		}
	})
	s.Register(ctx, transactionLifecycleRecord{})
	tx, err := s.Begin(ctx, true)
	if err != nil {
		testutil.ErrorIf(t, true, "begin transaction: %v", err)
	}
	if _, ok := transactionLocks.Load(tx); !ok {
		testutil.ErrorIf(t, true, "%v", "transaction lock was not registered")
	}
	if err := tx.Insert(&transactionLifecycleRecord{Name: "committed"}); err != nil {
		testutil.ErrorIf(t, true, "insert record: %v", err)
	}

	if err := s.Commit(tx); err != nil {
		testutil.ErrorIf(t, true, "commit transaction: %v", err)
	}
	if _, ok := transactionLocks.Load(tx); ok {
		testutil.ErrorIf(t, true, "%v", "transaction lock remained registered after commit")
	}

	var records []transactionLifecycleRecord
	if err := s.Read(ctx, func(tx *bstore.Tx) error {
		var err error
		records, err = bstore.QueryTx[transactionLifecycleRecord](tx).List()
		return err
	}); err != nil {
		testutil.ErrorIf(t, true, "read records: %v", err)
	}
	if len(records) != 1 || records[0].Name != "committed" {
		testutil.ErrorIf(t, true, "records = %#v, want committed record", records)
	}
}
