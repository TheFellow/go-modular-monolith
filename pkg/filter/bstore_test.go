package filter_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/filter"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
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
	testutil.Ok(t, err)
	t.Cleanup(func() {
		testutil.Ok(t, db.Close())
	})
	for _, r := range []timedRow{
		{CreatedAt: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)},
		{CreatedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
	} {
		testutil.Ok(t, db.Insert(ctx, &r))
	}
	expression, err := filter.Parse(filter.NewSchema[timedView](), `created_at >= date("2026-07-01T00:00:00Z")`)
	testutil.Ok(t, err)
	q := filter.ApplyBstore(bstore.QueryDB[timedRow](ctx, db), expression, func(r timedRow) timedView {
		return timedView{CreatedAt: r.CreatedAt}
	})
	rows, err := q.List()
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(rows) != 1 || rows[0].CreatedAt.Month() != time.August, "rows = %#v", rows)
}

func TestApplyBstoreCombinesPushdownAndArbitraryBooleanResidual(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := bstore.Open(ctx, filepath.Join(t.TempDir(), "filter.db"), nil, row{})
	testutil.Ok(t, err)
	t.Cleanup(func() {
		testutil.Ok(t, db.Close())
	})
	for _, r := range []row{
		{Name: "London gin", Category: "spirit"},
		{Name: "Old rum", Category: "spirit", Deleted: true},
		{Name: "Ginger beer", Category: "mixer"},
	} {
		testutil.Ok(t, db.Insert(ctx, &r))
	}

	schema := filter.NewSchema[view]()
	expression, err := filter.Parse(schema, `category == "spirit" && (name.contains("gin") || !deleted)`)
	testutil.Ok(t, err)
	q := bstore.QueryDB[row](ctx, db)
	q = filter.ApplyBstore(q, expression, func(r row) view {
		return view{Name: r.Name, Category: r.Category, Deleted: r.Deleted}
	})
	rows, err := q.List()
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(rows) != 1 || rows[0].Name != "London gin", "rows = %#v", rows)
}

func TestApplyBstoreDoesNotPushUnsafeOr(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := bstore.Open(ctx, filepath.Join(t.TempDir(), "filter.db"), nil, row{})
	testutil.Ok(t, err)
	t.Cleanup(func() {
		testutil.Ok(t, db.Close())
	})
	for _, r := range []row{{Name: "Gin", Category: "spirit"}, {Name: "Beer", Category: "mixer"}} {
		testutil.Ok(t, db.Insert(ctx, &r))
	}
	expression, err := filter.Parse(filter.NewSchema[view](), `category == "spirit" || name == "Beer"`)
	testutil.Ok(t, err)
	q := filter.ApplyBstore(bstore.QueryDB[row](ctx, db), expression, func(r row) view {
		return view{Name: r.Name, Category: r.Category}
	})
	rows, err := q.List()
	testutil.Ok(t, err)
	testutil.Equals(t, len(rows), 2)
}
