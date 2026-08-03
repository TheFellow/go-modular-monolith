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
	Score    int
}

type pushdownView struct {
	Name     string `expr:"name" filter:"Display name" filter-column:"Name"`
	Category string `expr:"category" filter:"Category" filter-column:"Category"`
	Deleted  bool   `expr:"deleted" filter:"Whether deleted" filter-column:"Deleted"`
	Score    int    `expr:"score" filter:"Score" filter-column:"Score"`
}

type timedRow struct {
	ID        int
	CreatedAt time.Time `bstore:"index"`
}

type timedView struct {
	CreatedAt time.Time `expr:"created_at" filter:"Creation time" filter-column:"CreatedAt"`
}

type taggedView struct {
	Category string   `expr:"category" filter:"Category" filter-column:"Category"`
	Tags     []string `expr:"tags" filter:"Tags"`
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

func TestApplyBstorePushdownsDefersCompleteEvaluation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := bstore.Open(ctx, filepath.Join(t.TempDir(), "filter.db"), nil, row{})
	testutil.Ok(t, err)
	t.Cleanup(func() { testutil.Ok(t, db.Close()) })
	for _, r := range []row{
		{Name: "Gin", Category: "spirit"},
		{Name: "Rum", Category: "spirit"},
		{Name: "Beer", Category: "mixer"},
	} {
		testutil.Ok(t, db.Insert(ctx, &r))
	}

	expression, err := filter.Parse(filter.NewSchema[taggedView](), `category == "spirit" && tags contains "featured"`)
	testutil.Ok(t, err)
	rows, err := filter.ApplyBstorePushdowns(bstore.QueryDB[row](ctx, db), expression).List()
	testutil.Ok(t, err)
	// The persisted conjunct is pushed, while the tag-dependent residual is
	// deliberately left for evaluation after callers hydrate tags.
	testutil.Equals(t, len(rows), 2)

	matched, err := expression.Match(taggedView{Category: rows[0].Category, Tags: []string{"featured"}})
	testutil.Ok(t, err)
	testutil.IsTrue(t, matched)
	matched, err = expression.Match(taggedView{Category: rows[1].Category})
	testutil.Ok(t, err)
	testutil.IsFalse(t, matched)
}

func TestApplyBstorePushdownsMappedBooleanFields(t *testing.T) {
	t.Parallel()

	rows := []row{
		{Name: "visible", Deleted: false},
		{Name: "deleted", Deleted: true},
	}
	for _, test := range []struct {
		source string
		want   []string
	}{
		{source: `deleted`, want: []string{"deleted"}},
		{source: `!deleted`, want: []string{"visible"}},
		{source: `!!deleted`, want: []string{"deleted"}},
	} {
		test := test
		t.Run(test.source, func(t *testing.T) {
			t.Parallel()
			got := pushdownCandidateNames(t, rows, test.source)
			testutil.Equals(t, got, test.want)
		})
	}
}

func TestApplyBstorePushdownsNegatedComparisons(t *testing.T) {
	t.Parallel()

	rows := []row{
		{Name: "one", Category: "spirit", Score: 1},
		{Name: "two", Category: "mixer", Score: 2},
		{Name: "three", Category: "other", Score: 3},
	}
	for _, test := range []struct {
		source string
		want   []string
	}{
		{source: `!(score == 2)`, want: []string{"one", "three"}},
		{source: `!(score != 2)`, want: []string{"two"}},
		{source: `!(score > 2)`, want: []string{"one", "two"}},
		{source: `!(score >= 2)`, want: []string{"one"}},
		{source: `!(score < 2)`, want: []string{"two", "three"}},
		{source: `!(score <= 2)`, want: []string{"three"}},
		{source: `!(2 < score)`, want: []string{"one", "two"}},
		{source: `!(category in ["spirit", "mixer"])`, want: []string{"three"}},
		{source: `!(category not in ["spirit", "mixer"])`, want: []string{"one", "two"}},
	} {
		test := test
		t.Run(test.source, func(t *testing.T) {
			t.Parallel()
			got := pushdownCandidateNames(t, rows, test.source)
			testutil.Equals(t, got, test.want)
		})
	}
}

func TestApplyBstorePushdownsAcrossBooleanGroups(t *testing.T) {
	t.Parallel()

	rows := []row{
		{Name: "gin", Category: "spirit"},
		{Name: "beer", Category: "mixer"},
		{Name: "cherry", Category: "garnish"},
	}
	for _, test := range []struct {
		name   string
		source string
		want   []string
	}{
		{
			name:   "equality OR becomes multi-value equality",
			source: `category == "spirit" || category == "mixer"`,
			want:   []string{"gin", "beer"},
		},
		{
			name:   "in and equality OR combine",
			source: `category in ["spirit"] || category == "mixer"`,
			want:   []string{"gin", "beer"},
		},
		{
			name:   "negated equality OR becomes multi-value not-equality",
			source: `!(category == "spirit" || category == "mixer")`,
			want:   []string{"cherry"},
		},
		{
			name:   "negated not-equality AND becomes multi-value equality",
			source: `!(category != "spirit" && category != "mixer")`,
			want:   []string{"gin", "beer"},
		},
		{
			name:   "different OR inequalities are not narrowed",
			source: `category != "spirit" || category != "mixer"`,
			want:   []string{"gin", "beer", "cherry"},
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := pushdownCandidateNames(t, rows, test.source)
			testutil.Equals(t, got, test.want)
		})
	}
}

func TestApplyBstorePushdownsExtractsNecessaryORConstraints(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := bstore.Open(ctx, filepath.Join(t.TempDir(), "filter.db"), nil, row{})
	testutil.Ok(t, err)
	t.Cleanup(func() { testutil.Ok(t, db.Close()) })
	for _, r := range []row{
		{Name: "gin", Category: "spirit"},
		{Name: "beer", Category: "mixer"},
		{Name: "cherry", Category: "garnish"},
	} {
		testutil.Ok(t, db.Insert(ctx, &r))
	}

	// Each branch requires one of the two persisted categories, even though
	// the tag predicates themselves must wait for hydration.
	expression, err := filter.Parse(filter.NewSchema[taggedView](),
		`(category == "spirit" && tags contains "featured") || (category == "mixer" && tags contains "seasonal")`)
	testutil.Ok(t, err)
	got, err := filter.ApplyBstorePushdowns(bstore.QueryDB[row](ctx, db), expression).List()
	testutil.Ok(t, err)
	testutil.Equals(t, rowNames(got), []string{"gin", "beer"})

	// An identical persisted predicate is also safe to extract from both OR
	// branches while their differing residual predicates remain deferred.
	expression, err = filter.Parse(filter.NewSchema[taggedView](),
		`(category == "spirit" && tags contains "featured") || (category == "spirit" && tags contains "seasonal")`)
	testutil.Ok(t, err)
	got, err = filter.ApplyBstorePushdowns(bstore.QueryDB[row](ctx, db), expression).List()
	testutil.Ok(t, err)
	testutil.Equals(t, rowNames(got), []string{"gin"})
}

func pushdownCandidateNames(t *testing.T, rows []row, source string) []string {
	t.Helper()
	ctx := context.Background()
	db, err := bstore.Open(ctx, filepath.Join(t.TempDir(), "filter.db"), nil, row{})
	testutil.Ok(t, err)
	t.Cleanup(func() { testutil.Ok(t, db.Close()) })
	for _, r := range rows {
		testutil.Ok(t, db.Insert(ctx, &r))
	}
	expression, err := filter.Parse(filter.NewSchema[pushdownView](), source)
	testutil.Ok(t, err)
	got, err := filter.ApplyBstorePushdowns(bstore.QueryDB[row](ctx, db), expression).List()
	testutil.Ok(t, err)
	return rowNames(got)
}

func rowNames(rows []row) []string {
	names := make([]string, len(rows))
	for i, row := range rows {
		names[i] = row.Name
	}
	return names
}
