package filter_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/filter"
	"github.com/mjl-/bstore"
)

type row struct {
	ID       int
	Name     string `bstore:"index"`
	Category string `bstore:"index"`
	Deleted  bool
}

type timedRow struct {
	ID        int
	CreatedAt time.Time `bstore:"index"`
}

type timedView struct {
	CreatedAt time.Time `expr:"created_at" filter:"Creation time" filter-column:"CreatedAt"`
}

func TestApplyBstorePushesCheckedDateLiteral(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := bstore.Open(ctx, filepath.Join(t.TempDir(), "filter.db"), nil, timedRow{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	for _, r := range []timedRow{
		{CreatedAt: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)},
		{CreatedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
	} {
		if err := db.Insert(ctx, &r); err != nil {
			t.Fatal(err)
		}
	}
	expression, err := filter.Parse(filter.NewSchema[timedView](), `created_at >= date("2026-07-01T00:00:00Z")`)
	if err != nil {
		t.Fatal(err)
	}
	q := filter.ApplyBstore(bstore.QueryDB[timedRow](ctx, db), expression, func(r timedRow) timedView {
		return timedView{CreatedAt: r.CreatedAt}
	})
	rows, err := q.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].CreatedAt.Month() != time.August {
		t.Fatalf("rows = %#v", rows)
	}
}

func TestApplyBstoreCombinesPushdownAndArbitraryBooleanResidual(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := bstore.Open(ctx, filepath.Join(t.TempDir(), "filter.db"), nil, row{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	for _, r := range []row{
		{Name: "London gin", Category: "spirit"},
		{Name: "Old rum", Category: "spirit", Deleted: true},
		{Name: "Ginger beer", Category: "mixer"},
	} {
		if err := db.Insert(ctx, &r); err != nil {
			t.Fatal(err)
		}
	}

	schema := filter.NewSchema[view]()
	expression, err := filter.Parse(schema, `category == "spirit" && (name.contains("gin") || !deleted)`)
	if err != nil {
		t.Fatal(err)
	}
	q := bstore.QueryDB[row](ctx, db)
	q = filter.ApplyBstore(q, expression, func(r row) view {
		return view{Name: r.Name, Category: r.Category, Deleted: r.Deleted}
	})
	rows, err := q.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Name != "London gin" {
		t.Fatalf("rows = %#v", rows)
	}
}

func TestApplyBstoreDoesNotPushUnsafeOr(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := bstore.Open(ctx, filepath.Join(t.TempDir(), "filter.db"), nil, row{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	for _, r := range []row{{Name: "Gin", Category: "spirit"}, {Name: "Beer", Category: "mixer"}} {
		if err := db.Insert(ctx, &r); err != nil {
			t.Fatal(err)
		}
	}
	expression, err := filter.Parse(filter.NewSchema[view](), `category == "spirit" || name == "Beer"`)
	if err != nil {
		t.Fatal(err)
	}
	q := filter.ApplyBstore(bstore.QueryDB[row](ctx, db), expression, func(r row) view {
		return view{Name: r.Name, Category: r.Category}
	})
	rows, err := q.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows", len(rows))
	}
}
