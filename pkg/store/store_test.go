package store

import (
	"context"
	"path/filepath"
	"testing"

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
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	s.Register(ctx, transactionLifecycleRecord{})
	tx, err := s.Begin(ctx, true)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	if _, ok := transactionLocks.Load(tx); !ok {
		t.Fatal("transaction lock was not registered")
	}
	if err := tx.Insert(&transactionLifecycleRecord{Name: "committed"}); err != nil {
		t.Fatalf("insert record: %v", err)
	}

	if err := s.Commit(tx); err != nil {
		t.Fatalf("commit transaction: %v", err)
	}
	if _, ok := transactionLocks.Load(tx); ok {
		t.Fatal("transaction lock remained registered after commit")
	}

	var records []transactionLifecycleRecord
	if err := s.Read(ctx, func(tx *bstore.Tx) error {
		var err error
		records, err = bstore.QueryTx[transactionLifecycleRecord](tx).List()
		return err
	}); err != nil {
		t.Fatalf("read records: %v", err)
	}
	if len(records) != 1 || records[0].Name != "committed" {
		t.Fatalf("records = %#v, want committed record", records)
	}
}
