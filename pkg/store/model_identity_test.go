package store

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	testutil "github.com/TheFellow/go-modular-monolith/pkg/testutil/assert"
)

type relocatedRecord revisionedRecord

func (relocatedRecord) StoreModelName() string { return modelName(reflect.TypeFor[revisionedRecord]()) }

func TestRelocatedModelPreservesExistingRecordsAndRevisions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "relocated.db"))
	testutil.ErrorIf(t, err != nil, "unexpected error: %v", err)
	defer func() { _ = s.Close() }()
	original := revisionedRecord{ID: 1, Name: "before"}
	ok(t, s.Write(ctx, func(tx *Tx) error { return tx.Insert(&original) }))
	s.Register(ctx, relocatedRecord{})
	moved := relocatedRecord{ID: 1}
	ok(t, s.Read(ctx, func(tx *Tx) error { return tx.Get(&moved) }))
	testutil.ErrorIf(t, moved.Name != original.Name, "name changed")
	testutil.ErrorIf(t, moved.Revision != original.Revision, "revision changed")
	moved.Name = "after"
	ok(t, s.Write(ctx, func(tx *Tx) error { return tx.Update(&moved) }))
	ok(t, s.Read(ctx, func(tx *Tx) error { return tx.Get(&original) }))
	testutil.ErrorIf(t, original.Name != "after", "update lost")
	testutil.ErrorIf(t, original.Revision != moved.Revision, "revision lost")
}

func ok(t *testing.T, err error) {
	t.Helper()
	testutil.ErrorIf(t, err != nil, "unexpected error: %v", err)
}
