package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	testutil "github.com/TheFellow/go-modular-monolith/pkg/testutil/assert"
)

type transactionLifecycleRecord struct {
	ID   int
	Name string
}

type timeQueryRecord struct {
	ID int
	At time.Time `store:"index"`
}

type revisionedRecord struct {
	ID       int
	Revision uint64 `json:"-" store:"revision"`
	Name     string
}

func TestOptimisticRevisionRejectsStaleUpdate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "revisions.db")
	first, err := Open(ctx, path)
	testutil.ErrorIf(t, err != nil, "open first store: %v", err)
	defer func() { _ = first.Close() }()
	second, err := Open(ctx, path)
	testutil.ErrorIf(t, err != nil, "open second store: %v", err)
	defer func() { _ = second.Close() }()

	record := revisionedRecord{ID: 1, Name: "original"}
	err = first.Write(ctx, func(tx *Tx) error { return tx.Insert(&record) })
	testutil.ErrorIf(t, err != nil || record.Revision != 1, "insert revision = %d, err = %v", record.Revision, err)

	left, right := revisionedRecord{ID: 1}, revisionedRecord{ID: 1}
	err = first.Read(ctx, func(tx *Tx) error { return tx.Get(&left) })
	testutil.ErrorIf(t, err != nil, "read left: %v", err)
	err = second.Read(ctx, func(tx *Tx) error { return tx.Get(&right) })
	testutil.ErrorIf(t, err != nil, "read right: %v", err)

	left.Name = "winner"
	err = first.Write(ctx, func(tx *Tx) error { return tx.Update(&left) })
	testutil.ErrorIf(t, err != nil || left.Revision != 2, "winning revision = %d, err = %v", left.Revision, err)

	right.Name = "stale"
	err = second.Write(ctx, func(tx *Tx) error { return tx.Update(&right) })
	testutil.ErrorIf(t, !apperrors.IsConflict(err), "stale update error = %v, want conflict", err)
	err = second.Write(ctx, func(tx *Tx) error { return tx.Delete(&right) })
	testutil.ErrorIf(t, !apperrors.IsConflict(err), "stale delete error = %v, want conflict", err)

	stored := revisionedRecord{ID: 1}
	err = second.Read(ctx, func(tx *Tx) error { return tx.Get(&stored) })
	testutil.ErrorIf(t, err != nil || stored.Name != "winner" || stored.Revision != 2, "stored = %#v, err = %v", stored, err)
}

func TestTimeMultiValuePredicatesAndOrdering(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "times.db"))
	testutil.ErrorIf(t, err != nil, "open store: %v", err)
	defer func() { _ = s.Close() }()
	s.Register(ctx, timeQueryRecord{})
	early := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).In(time.FixedZone("plus14", 14*60*60))
	middle := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC).In(time.FixedZone("minus10", -10*60*60))
	late := time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC)
	testutil.ErrorIf(t, s.Write(ctx, func(tx *Tx) error {
		return tx.Insert(&timeQueryRecord{1, early}, &timeQueryRecord{2, middle}, &timeQueryRecord{3, late})
	}) != nil, "insert time rows")
	var filtered, ordered []timeQueryRecord
	err = s.Read(ctx, func(tx *Tx) error {
		filtered, err = QueryTx[timeQueryRecord](tx).FilterEqual("At", early, late).SortAsc("At").List()
		if err != nil {
			return err
		}
		ordered, err = QueryTx[timeQueryRecord](tx).SortAsc("At").List()
		return err
	})
	testutil.ErrorIf(t, err != nil, "query time rows: %v", err)
	testutil.ErrorIf(t, len(filtered) != 2 || filtered[0].ID != 1 || filtered[1].ID != 3, "filtered = %#v", filtered)
	testutil.ErrorIf(t, len(ordered) != 3 || ordered[0].ID != 1 || ordered[1].ID != 2 || ordered[2].ID != 3, "ordered = %#v", ordered)
}

func TestMigrationVersionBookkeepingAndFutureVersion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "versions.db")
	s, err := Open(ctx, path)
	testutil.ErrorIf(t, err != nil, "open store: %v", err)
	var version, count int
	err = s.db.QueryRowContext(ctx, "SELECT MAX(version), COUNT(*) FROM schema_migrations").Scan(&version, &count)
	testutil.ErrorIf(t, err != nil, "read versions: %v", err)
	testutil.ErrorIf(t, version != len(schemaMigrations) || count != len(schemaMigrations), "version/count = %d/%d", version, count)
	_, err = s.db.ExecContext(ctx, "INSERT INTO schema_migrations(version, applied_at) VALUES (?, 'future')", len(schemaMigrations)+1)
	testutil.ErrorIf(t, err != nil, "insert future version: %v", err)
	testutil.ErrorIf(t, s.Close() != nil, "close version store")
	newer, err := Open(ctx, path)
	if newer != nil {
		_ = newer.Close()
	}
	testutil.ErrorIf(t, err == nil, "opening a future schema version unexpectedly succeeded")
}

func TestRevisionMigrationUpgradesExistingRows(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "revision-upgrade.db")
	legacy, err := sql.Open("sqlite", path)
	testutil.ErrorIf(t, err != nil, "open legacy database: %v", err)
	_, err = legacy.ExecContext(ctx, `CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`)
	testutil.ErrorIf(t, err != nil, "create legacy migration ledger: %v", err)
	_, err = legacy.ExecContext(ctx, `CREATE TABLE records (model TEXT NOT NULL, id TEXT NOT NULL, data TEXT NOT NULL CHECK(json_valid(data)), PRIMARY KEY(model, id))`)
	testutil.ErrorIf(t, err != nil, "create legacy records: %v", err)
	_, err = legacy.ExecContext(ctx, `INSERT INTO schema_migrations(version, applied_at) VALUES (1, 'legacy')`)
	testutil.ErrorIf(t, err != nil, "record legacy migration: %v", err)
	_, err = legacy.ExecContext(ctx, `INSERT INTO records(model,id,data) VALUES (?, '1', '{"ID":1,"Name":"legacy"}')`, modelName(reflect.TypeFor[revisionedRecord]()))
	testutil.ErrorIf(t, err != nil, "insert legacy row: %v", err)
	testutil.ErrorIf(t, legacy.Close() != nil, "close legacy database")

	upgraded, err := Open(ctx, path)
	testutil.ErrorIf(t, err != nil, "upgrade database: %v", err)
	defer func() { _ = upgraded.Close() }()
	record := revisionedRecord{ID: 1}
	err = upgraded.Read(ctx, func(tx *Tx) error { return tx.Get(&record) })
	testutil.ErrorIf(t, err != nil || record.Name != "legacy" || record.Revision != 1, "upgraded row = %#v, err = %v", record, err)
}

func TestConcurrentMigrationInitialization(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "concurrent-migrations.db")
	errCh := make(chan error, 8)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			s, err := Open(ctx, path)
			if err == nil {
				err = s.Close()
			}
			errCh <- err
		})
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		testutil.ErrorIf(t, err != nil, "concurrent initialization: %v", err)
	}
	s, err := Open(ctx, path)
	testutil.ErrorIf(t, err != nil, "reopen initialized store: %v", err)
	defer func() { _ = s.Close() }()
	var count int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	testutil.ErrorIf(t, err != nil || count != len(schemaMigrations), "migration count = %d, err = %v", count, err)
}

func TestSQLStringLiteral(t *testing.T) {
	t.Parallel()
	got := sqlStringLiteral("owner's.model")
	testutil.ErrorIf(t, got != "'owner''s.model'", "literal = %s", got)
}

func TestIndependentStoresShareOneDatabase(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "shared.db")
	first, err := Open(ctx, path)
	testutil.ErrorIf(t, err != nil, "open first store: %v", err)
	defer func() { _ = first.Close() }()
	second, err := Open(ctx, path)
	testutil.ErrorIf(t, err != nil, "open second store: %v", err)
	defer func() { _ = second.Close() }()
	first.Register(ctx, transactionLifecycleRecord{})
	second.Register(ctx, transactionLifecycleRecord{})

	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := range 20 {
		s := first
		if i%2 == 1 {
			s = second
		}
		wg.Go(func() {
			errs <- s.Write(ctx, func(tx *Tx) error {
				return tx.Insert(&transactionLifecycleRecord{ID: i + 1, Name: strconv.Itoa(i)})
			})
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		testutil.ErrorIf(t, err != nil, "concurrent write: %v", err)
	}

	var rows []transactionLifecycleRecord
	err = second.Read(ctx, func(tx *Tx) error {
		rows, err = QueryTx[transactionLifecycleRecord](tx).List()
		return err
	})
	testutil.ErrorIf(t, err != nil, "read shared rows: %v", err)
	testutil.ErrorIf(t, len(rows) != 20, "shared rows = %d, want 20", len(rows))
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
		err := s.Read(ctx, func(tx *Tx) error {
			var err error
			records, err = QueryTx[transactionLifecycleRecord](tx).List()
			return err
		})
		testutil.ErrorIf(t, err != nil, "read records: %v", err)
	}
	testutil.ErrorIf(t, len(records) != 1 || records[0].Name != "committed", "records = %#v, want committed record", records)
}
