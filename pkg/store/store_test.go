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
	testutil.ErrorIf(t, err != nil, "open store: %v", err)
	t.Cleanup(func() {
		{
			err := s.Close()
			testutil.ErrorIf(t, err != nil, "close store: %v", err)
		}
	})
	s.Register(ctx, transactionLifecycleRecord{})
	tx, err := s.Begin(ctx, true)
	testutil.ErrorIf(t, err != nil, "begin transaction: %v", err)
	{
		_, ok := transactionLocks.Load(tx)
		testutil.ErrorIf(t, !ok, "%v", "transaction lock was not registered")
	}
	{
		err := tx.Insert(&transactionLifecycleRecord{Name: "committed"})
		testutil.ErrorIf(t, err != nil, "insert record: %v", err)
	}

	{
		err := s.Commit(tx)
		testutil.ErrorIf(t, err != nil, "commit transaction: %v", err)
	}
	{
		_, ok := transactionLocks.Load(tx)
		testutil.ErrorIf(t, ok, "%v", "transaction lock remained registered after commit")
	}

	var records []transactionLifecycleRecord
	{
		err := s.Read(ctx, func(tx *bstore.Tx) error {
			var err error
			records, err = bstore.QueryTx[transactionLifecycleRecord](tx).List()
			return err
		})
		testutil.ErrorIf(t, err != nil, "read records: %v", err)
	}
	testutil.ErrorIf(t, len(records) != 1 || records[0].Name != "committed", "records = %#v, want committed record", records)
}
