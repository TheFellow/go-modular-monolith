package store

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	testutil "github.com/TheFellow/go-modular-monolith/pkg/testutil/assert"
	"github.com/govalues/decimal"
)

type relationalLine struct {
	Name         string
	Quantity     int
	Alternatives []string
}

type relationalRecord struct {
	ID       int
	Revision uint64    `store:"revision"`
	Owner    string    `store:"index=Owner+At"`
	At       time.Time `store:"index"`
	Active   bool
	Price    struct {
		Amount   decimal.Decimal
		Currency string
	}
	Lines   []relationalLine
	Labels  map[string]string
	Empty   []string
	Nil     []string
	Pointer *relationalLine
	Maybe   optional.Value[relationalLine]
}

func (relationalRecord) StoreModelName() string { return "relational_records" }

func sampleRelationalRecord(t *testing.T) relationalRecord {
	t.Helper()
	record := relationalRecord{
		ID: 7, Owner: "owner", Active: true,
		At:     time.Date(2026, 9, 22, 11, 4, 3, 123456789, time.UTC),
		Lines:  []relationalLine{{Name: "first", Quantity: 2, Alternatives: []string{"a", "b"}}, {Name: "second", Quantity: 3, Alternatives: []string{}}},
		Labels: map[string]string{"quoted ' key": "value", "second": "two"}, Empty: []string{},
	}
	var err error
	record.Price.Amount, err = decimal.Parse("1234567890123456.789")
	ok(t, err)
	record.Price.Currency = "USD"
	return record
}

func TestRelationalCollectionsTransactionsAndReopen(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "relations.db")
	s, err := Open(ctx, path)
	ok(t, err)
	defer func() { _ = s.Close() }()
	s.Register(ctx, relationalRecord{})
	original := sampleRelationalRecord(t)
	ok(t, s.Write(ctx, func(tx *Tx) error { return tx.Insert(&original) }))
	assertStored := func(want relationalRecord) {
		t.Helper()
		got := relationalRecord{ID: want.ID}
		ok(t, s.Read(ctx, func(tx *Tx) error { return tx.Get(&got) }))
		if !reflect.DeepEqual(got, want) {
			testutil.ErrorIf(t, true, "round trip mismatch:\n got %#v\nwant %#v", got, want)
		}
	}
	assertStored(original)
	aborted := sampleRelationalRecord(t)
	aborted.Revision = original.Revision
	aborted.Lines = []relationalLine{{Name: "rollback", Alternatives: []string{"never committed"}}}
	aborted.Labels = nil
	sentinel := errors.New("abort transaction")
	err = s.Write(ctx, func(tx *Tx) error {
		if err := tx.Update(&aborted); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		testutil.ErrorIf(t, true, "rollback error = %v", err)
	}
	assertStored(original)
	updated := sampleRelationalRecord(t)
	updated.Revision = original.Revision
	updated.Lines = []relationalLine{{Name: "replacement", Quantity: 4, Alternatives: []string{"c"}}}
	updated.Labels = map[string]string{}
	updated.Empty = nil
	updated.Nil = []string{}
	updated.Pointer = &relationalLine{Name: "pointer", Alternatives: []string{"nested"}}
	updated.Maybe = optional.Some(relationalLine{Name: "optional", Alternatives: []string{"nested optional"}})
	ok(t, s.Write(ctx, func(tx *Tx) error { return tx.Update(&updated) }))
	if updated.Revision != original.Revision+1 {
		testutil.ErrorIf(t, true, "revision = %d", updated.Revision)
	}
	assertStored(updated)
	err = s.Write(ctx, func(tx *Tx) error { return tx.Update(&original) })
	if !apperrors.IsConflict(err) {
		testutil.ErrorIf(t, true, "stale collection update = %v", err)
	}
	assertStored(updated)
	ok(t, s.Close())
	s, err = Open(ctx, path)
	ok(t, err)
	s.Register(ctx, relationalRecord{})
	assertStored(updated)
	ok(t, s.Write(ctx, func(tx *Tx) error { return tx.Delete(&updated) }))
	for _, table := range []string{"relational_records", "relational_records_lines", "relational_records_lines_alternatives", "relational_records_labels"} {
		var count int
		ok(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM "`+table+`"`).Scan(&count))
		if count != 0 {
			testutil.ErrorIf(t, true, "delete left %d rows in %s", count, table)
		}
	}
}

func TestRelationalSchemaConstraintsAndQueryPlans(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "schema.db"))
	ok(t, err)
	defer func() { _ = s.Close() }()
	s.Register(ctx, relationalRecord{})
	record := sampleRelationalRecord(t)
	ok(t, s.Write(ctx, func(tx *Tx) error { return tx.Insert(&record) }))
	var legacyTables int
	ok(t, s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('records','sequences')").Scan(&legacyTables))
	if legacyTables != 0 {
		testutil.ErrorIf(t, true, "generic document tables remain")
	}
	var idType, amountType, activeType, amount string
	ok(t, s.db.QueryRowContext(ctx, "SELECT typeof(id), typeof(price_amount), typeof(active), price_amount FROM relational_records WHERE id=7").Scan(&idType, &amountType, &activeType, &amount))
	if idType != "integer" || amountType != "text" || activeType != "integer" || amount != "1234567890123456.789" {
		testutil.ErrorIf(t, true, "typed relational values = %q %q %q %q", idType, amountType, activeType, amount)
	}
	var schema string
	ok(t, s.db.QueryRowContext(ctx, "SELECT sql FROM sqlite_master WHERE name='relational_records_lines'").Scan(&schema))
	if !strings.Contains(strings.ToUpper(schema), "ON DELETE CASCADE") {
		testutil.ErrorIf(t, true, "child lacks cascading FK: %s", schema)
	}
	_, err = s.db.ExecContext(ctx, "UPDATE relational_records_lines SET __parent_id=999999 WHERE __parent_id=7")
	if err == nil {
		testutil.ErrorIf(t, true, "orphaned child row accepted")
	}
	_, err = s.db.ExecContext(ctx, "UPDATE relational_records_lines SET __position=0 WHERE __parent_id=7 AND __position=1")
	testutil.ErrorIf(t, err == nil, "duplicate child collection position accepted")
	_, err = s.db.ExecContext(ctx, "UPDATE relational_records_labels SET key='second' WHERE key=?", "quoted ' key")
	testutil.ErrorIf(t, err == nil, "duplicate map key accepted")
	var violations int
	rows, err := s.db.QueryContext(ctx, "PRAGMA foreign_key_check")
	ok(t, err)
	for rows.Next() {
		violations++
	}
	ok(t, rows.Err())
	ok(t, rows.Close())
	testutil.ErrorIf(t, violations != 0, "foreign key violations: %d", violations)
	for _, query := range []string{
		"SELECT id FROM relational_records WHERE owner='owner' ORDER BY at",
		"SELECT id FROM relational_records WHERE at > '2026' ORDER BY at",
		"SELECT * FROM relational_records_lines WHERE __parent_id=7 ORDER BY __position",
	} {
		rows, err := s.db.QueryContext(ctx, "EXPLAIN QUERY PLAN "+query)
		ok(t, err)
		var plan strings.Builder
		for rows.Next() {
			var id, parent, unused int
			var detail string
			ok(t, rows.Scan(&id, &parent, &unused, &detail))
			plan.WriteString(detail)
		}
		ok(t, rows.Err())
		ok(t, rows.Close())
		description := strings.ToUpper(plan.String())
		if !strings.Contains(description, "SEARCH") || !strings.Contains(description, "INDEX") || strings.Contains(description, "TEMP B-TREE") {
			testutil.ErrorIf(t, true, "query lacks usable ordered index: %s: %s", query, plan.String())
		}
	}
}

func TestTimeNanosecondPredicatesAndZeroValue(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "nanoseconds.db"))
	ok(t, err)
	defer func() { _ = s.Close() }()
	s.Register(ctx, timeQueryRecord{})
	instant := time.Date(2026, 9, 22, 0, 0, 0, 123456788, time.UTC)
	ok(t, s.Write(ctx, func(tx *Tx) error {
		return tx.Insert(&timeQueryRecord{ID: 1}, &timeQueryRecord{ID: 2, At: instant}, &timeQueryRecord{ID: 3, At: instant.Add(time.Nanosecond)})
	}))
	ok(t, s.Read(ctx, func(tx *Tx) error {
		rows, err := QueryTx[timeQueryRecord](tx).FilterEqual("At", instant.Add(time.Nanosecond).In(time.FixedZone("offset", 3600))).List()
		if err != nil {
			return err
		}
		if len(rows) != 1 || rows[0].ID != 3 {
			testutil.ErrorIf(t, true, "nanosecond predicate = %#v", rows)
		}
		rows, err = QueryTx[timeQueryRecord](tx).SortAsc("At").List()
		if err != nil {
			return err
		}
		if len(rows) != 3 || rows[0].ID != 1 || !rows[0].At.IsZero() || rows[1].ID != 2 || rows[2].ID != 3 {
			testutil.ErrorIf(t, true, "nanosecond order = %#v", rows)
		}
		return nil
	}))
}

func TestFailedChildMutationIsAtomicWhenCallerContinues(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "savepoint.db"))
	ok(t, err)
	defer func() { _ = s.Close() }()
	s.Register(ctx, relationalRecord{})
	original := sampleRelationalRecord(t)
	ok(t, s.Write(ctx, func(tx *Tx) error { return tx.Insert(&original) }))
	_, err = s.db.ExecContext(ctx, `CREATE TRIGGER reject_child BEFORE INSERT ON relational_records_lines_alternatives
 WHEN NEW.value='reject' BEGIN SELECT RAISE(ABORT, 'rejected child'); END`)
	ok(t, err)
	replacement := sampleRelationalRecord(t)
	replacement.Revision = original.Revision
	replacement.Owner = "must roll back"
	replacement.Lines = []relationalLine{{Name: "new", Alternatives: []string{"accepted first", "reject"}}}
	newRecord := sampleRelationalRecord(t)
	newRecord.ID = 8
	newRecord.Lines = replacement.Lines
	ok(t, s.Write(ctx, func(tx *Tx) error {
		updateErr := tx.Update(&replacement)
		testutil.ErrorIf(t, updateErr == nil, "update unexpectedly accepted rejected child")
		testutil.ErrorIf(t, replacement.Revision != original.Revision, "failed update advanced revision")
		insertErr := tx.Insert(&newRecord)
		testutil.ErrorIf(t, insertErr == nil, "insert unexpectedly accepted rejected child")
		return nil // Commit the surrounding transaction after handling both failures.
	}))
	got := relationalRecord{ID: original.ID}
	ok(t, s.Read(ctx, func(tx *Tx) error {
		if err := tx.Get(&got); err != nil {
			return err
		}
		missing := relationalRecord{ID: 8}
		err := tx.Get(&missing)
		testutil.ErrorIf(t, !apperrors.IsNotFound(err), "failed insert left parent row: %v", err)
		return nil
	}))
	testutil.ErrorIf(t, !reflect.DeepEqual(got, original), "failed update changed stored aggregate: %#v", got)
}

func TestQueriesRejectUnknownFields(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "invalid-fields.db"))
	ok(t, err)
	defer func() { _ = s.Close() }()
	s.Register(ctx, relationalRecord{})
	ok(t, s.Read(ctx, func(tx *Tx) error {
		_, err := QueryTx[relationalRecord](tx).FilterEqual("Missing", "value").List()
		testutil.ErrorIf(t, err == nil, "unknown filter field silently accepted")
		_, err = QueryTx[relationalRecord](tx).SortAsc("Missing").List()
		testutil.ErrorIf(t, err == nil, "unknown sort field silently accepted")
		_, err = QueryTx[relationalRecord](tx).FilterEqual("Owner\"; DROP TABLE relational_records;--", "value").List()
		testutil.ErrorIf(t, err == nil, "invalid field identifier silently accepted")
		return nil
	}))
}

func TestRelationalColumnsRejectMalformedScalarValues(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "typed-columns.db"))
	ok(t, err)
	defer func() { _ = s.Close() }()
	s.Register(ctx, relationalRecord{})
	original := sampleRelationalRecord(t)
	ok(t, s.Write(ctx, func(tx *Tx) error { return tx.Insert(&original) }))
	for _, statement := range []string{
		"UPDATE relational_records SET id='not-an-integer' WHERE id=7",
		"UPDATE relational_records SET active='not-a-bool' WHERE id=7",
		"UPDATE relational_records_lines SET quantity='not-an-integer' WHERE __parent_id=7",
	} {
		_, err := s.db.ExecContext(ctx, statement)
		testutil.ErrorIf(t, err == nil, "malformed scalar accepted: %s", statement)
	}
	got := relationalRecord{ID: original.ID}
	ok(t, s.Read(ctx, func(tx *Tx) error { return tx.Get(&got) }))
	testutil.ErrorIf(t, !reflect.DeepEqual(got, original), "malformed writes changed aggregate: %#v", got)
}

type nullableRecord struct {
	ID        int
	DeletedAt *time.Time
	Score     optional.Value[int]
	Names     *[]string
	Aliases   optional.Value[[]string]
}

func TestNullableScalarPredicatesAndOrdering(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "nullable.db"))
	ok(t, err)
	defer func() { _ = s.Close() }()
	s.Register(ctx, nullableRecord{})
	early := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	late := early.Add(time.Hour)
	ok(t, s.Write(ctx, func(tx *Tx) error {
		return tx.Insert(&nullableRecord{ID: 1}, &nullableRecord{ID: 2, DeletedAt: &early, Score: optional.Some(0)}, &nullableRecord{ID: 3, DeletedAt: &late, Score: optional.Some(1)})
	}))
	ok(t, s.Read(ctx, func(tx *Tx) error {
		cases := []struct {
			name  string
			query *Query[nullableRecord]
			ids   []int
		}{
			{"pointer less", QueryTx[nullableRecord](tx).FilterLess("DeletedAt", late), []int{2}},
			{"pointer unequal", QueryTx[nullableRecord](tx).FilterNotEqual("DeletedAt", late), []int{2}},
			{"pointer ascending", QueryTx[nullableRecord](tx).SortAsc("DeletedAt"), []int{1, 2, 3}},
			{"pointer descending", QueryTx[nullableRecord](tx).SortDesc("DeletedAt"), []int{3, 2, 1}},
			{"optional equal zero", QueryTx[nullableRecord](tx).FilterEqual("Score", 0), []int{2}},
			{"optional less", QueryTx[nullableRecord](tx).FilterLess("Score", 1), []int{2}},
			{"optional ascending", QueryTx[nullableRecord](tx).SortAsc("Score"), []int{1, 2, 3}},
			{"optional descending", QueryTx[nullableRecord](tx).SortDesc("Score"), []int{3, 2, 1}},
		}
		for _, tc := range cases {
			records, err := tc.query.List()
			if err != nil {
				return err
			}
			ids := make([]int, len(records))
			for i, record := range records {
				ids[i] = record.ID
			}
			testutil.ErrorIf(t, !reflect.DeepEqual(ids, tc.ids), "%s: IDs = %v, want %v", tc.name, ids, tc.ids)
		}
		return nil
	}))
}

func TestWrappedCollectionsPreserveAbsentNilAndEmpty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "wrapped-collections.db"))
	ok(t, err)
	defer func() { _ = s.Close() }()
	s.Register(ctx, nullableRecord{})
	var nilNames []string
	emptyNames := []string{}
	names := []string{"first", "second"}
	records := []nullableRecord{
		{ID: 1},
		{ID: 2, Names: &nilNames, Aliases: optional.Some[[]string](nil)},
		{ID: 3, Names: &emptyNames, Aliases: optional.Some([]string{})},
		{ID: 4, Names: &names, Aliases: optional.Some([]string{"alias"})},
	}
	ok(t, s.Write(ctx, func(tx *Tx) error {
		for i := range records {
			if err := tx.Insert(&records[i]); err != nil {
				return err
			}
		}
		return nil
	}))
	ok(t, s.Read(ctx, func(tx *Tx) error {
		for _, want := range records {
			got := nullableRecord{ID: want.ID}
			if err := tx.Get(&got); err != nil {
				return err
			}
			testutil.ErrorIf(t, !reflect.DeepEqual(got, want), "wrapped collections roundtrip: got %#v, want %#v", got, want)
		}
		return nil
	}))
}
