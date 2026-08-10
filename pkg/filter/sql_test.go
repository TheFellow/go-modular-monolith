package filter_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/filter"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

type sqlRow struct {
	ID             int
	Name, Category string
	Score          int
	Deleted        bool
}
type sqlView struct {
	Name     string `expr:"name" filter:"Name" filter-column:"Name"`
	Category string `expr:"category" filter:"Category" filter-column:"Category"`
	Score    int    `expr:"score" filter:"Score" filter-column:"Score"`
	Deleted  bool   `expr:"deleted" filter:"Deleted" filter-column:"Deleted"`
}

type taggedView struct {
	Category string   `expr:"category" filter:"Category" filter-column:"Category"`
	Tags     []string `expr:"tags" filter:"Tags"`
}

func TestApplySQLPushdownAndResidual(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s, err := store.Open(ctx, filepath.Join(t.TempDir(), "filter.db"))
	testutil.Ok(t, err)
	t.Cleanup(func() { testutil.Ok(t, s.Close()) })
	s.Register(ctx, sqlRow{})
	testutil.Ok(t, s.Write(ctx, func(tx *store.Tx) error {
		for _, row := range []sqlRow{{ID: 1, Name: "lime", Category: "fruit", Score: 1}, {ID: 2, Name: "lemon", Category: "fruit", Score: 3}, {ID: 3, Name: "salt", Category: "pantry", Score: 4}} {
			if err := tx.Insert(&row); err != nil {
				return err
			}
		}
		return nil
	}))
	expression, err := filter.Parse(filter.NewSchema[sqlView](), `category == "fruit" && score > 1`)
	testutil.Ok(t, err)
	var rows []sqlRow
	testutil.Ok(t, s.Read(ctx, func(tx *store.Tx) error {
		rows, err = filter.ApplySQL(store.QueryTx[sqlRow](tx), expression, func(r sqlRow) sqlView {
			return sqlView{Name: r.Name, Category: r.Category, Score: r.Score}
		}).List()
		return err
	}))
	testutil.ErrorIf(t, len(rows) != 1 || rows[0].Name != "lemon", "rows = %#v", rows)
}

func TestApplySQLPushdownBooleanSemantics(t *testing.T) {
	t.Parallel()
	rows := []sqlRow{
		{ID: 1, Name: "one", Category: "spirit", Score: 1},
		{ID: 2, Name: "two", Category: "mixer", Score: 2, Deleted: true},
		{ID: 3, Name: "three", Category: "other", Score: 3},
	}
	for _, test := range []struct {
		source string
		want   []string
	}{
		{`deleted`, []string{"two"}},
		{`!deleted`, []string{"one", "three"}},
		{`!!deleted`, []string{"two"}},
		{`!(score == 2)`, []string{"one", "three"}},
		{`!(score != 2)`, []string{"two"}},
		{`!(score > 2)`, []string{"one", "two"}},
		{`!(score >= 2)`, []string{"one"}},
		{`!(score < 2)`, []string{"two", "three"}},
		{`!(score <= 2)`, []string{"three"}},
		{`!(2 < score)`, []string{"one", "two"}},
		{`!(category in ["spirit", "mixer"])`, []string{"three"}},
		{`!(category not in ["spirit", "mixer"])`, []string{"one", "two"}},
		{`category == "spirit" || category == "mixer"`, []string{"one", "two"}},
		{`category in ["spirit"] || category == "mixer"`, []string{"one", "two"}},
		{`!(category == "spirit" || category == "mixer")`, []string{"three"}},
		{`!(category != "spirit" && category != "mixer")`, []string{"one", "two"}},
		{`category != "spirit" || category != "mixer"`, []string{"one", "two", "three"}},
	} {
		t.Run(test.source, func(t *testing.T) {
			t.Parallel()
			got := pushdownNames(t, rows, test.source)
			testutil.Equals(t, got, test.want)
		})
	}
}

func TestApplySQLCombinesPushdownWithArbitraryResidual(t *testing.T) {
	t.Parallel()
	rows := []sqlRow{
		{ID: 1, Name: "London gin", Category: "spirit"},
		{ID: 2, Name: "Old rum", Category: "spirit", Deleted: true},
		{ID: 3, Name: "Ginger beer", Category: "mixer"},
	}
	ctx, s := sqlStore(t, rows)
	expression, err := filter.Parse(filter.NewSchema[sqlView](), `category == "spirit" && (name.contains("gin") || !deleted)`)
	testutil.Ok(t, err)
	var got []sqlRow
	testutil.Ok(t, s.Read(ctx, func(tx *store.Tx) error {
		got, err = filter.ApplySQL(store.QueryTx[sqlRow](tx), expression, func(r sqlRow) sqlView {
			return sqlView{Name: r.Name, Category: r.Category, Score: r.Score, Deleted: r.Deleted}
		}).List()
		return err
	}))
	testutil.ErrorIf(t, len(got) != 1 || got[0].Name != "London gin", "rows = %#v", got)
}

func TestApplySQLDoesNotPushUnsafeOR(t *testing.T) {
	t.Parallel()
	rows := []sqlRow{{ID: 1, Name: "Gin", Category: "spirit"}, {ID: 2, Name: "Beer", Category: "mixer"}}
	ctx, s := sqlStore(t, rows)
	expression, err := filter.Parse(filter.NewSchema[sqlView](), `category == "spirit" || name == "Beer"`)
	testutil.Ok(t, err)
	var got []sqlRow
	testutil.Ok(t, s.Read(ctx, func(tx *store.Tx) error {
		got, err = filter.ApplySQL(store.QueryTx[sqlRow](tx), expression, func(r sqlRow) sqlView {
			return sqlView{Name: r.Name, Category: r.Category}
		}).List()
		return err
	}))
	testutil.Equals(t, len(got), 2)
}

func TestApplySQLPushdownsDeferHydratedFields(t *testing.T) {
	t.Parallel()
	rows := []sqlRow{{ID: 1, Name: "gin", Category: "spirit"}, {ID: 2, Name: "rum", Category: "spirit"}, {ID: 3, Name: "beer", Category: "mixer"}}
	ctx, s := sqlStore(t, rows)
	expression, err := filter.Parse(filter.NewSchema[taggedView](), `category == "spirit" && tags contains "featured"`)
	testutil.Ok(t, err)
	var candidates []sqlRow
	testutil.Ok(t, s.Read(ctx, func(tx *store.Tx) error {
		candidates, err = filter.ApplySQLPushdowns(store.QueryTx[sqlRow](tx), expression).List()
		return err
	}))
	testutil.Equals(t, len(candidates), 2)
	matched, err := expression.Match(taggedView{Category: candidates[0].Category, Tags: []string{"featured"}})
	testutil.Ok(t, err)
	testutil.IsTrue(t, matched)
}

func TestApplySQLPushdownsExtractNecessaryORConstraints(t *testing.T) {
	t.Parallel()
	rows := []sqlRow{{ID: 1, Name: "gin", Category: "spirit"}, {ID: 2, Name: "beer", Category: "mixer"}, {ID: 3, Name: "cherry", Category: "garnish"}}
	ctx, s := sqlStore(t, rows)
	for _, test := range []struct {
		source string
		want   []string
	}{
		{`(category == "spirit" && tags contains "featured") || (category == "mixer" && tags contains "seasonal")`, []string{"gin", "beer"}},
		{`(category == "spirit" && tags contains "featured") || (category == "spirit" && tags contains "seasonal")`, []string{"gin"}},
	} {
		expression, err := filter.Parse(filter.NewSchema[taggedView](), test.source)
		testutil.Ok(t, err)
		var got []sqlRow
		testutil.Ok(t, s.Read(ctx, func(tx *store.Tx) error {
			got, err = filter.ApplySQLPushdowns(store.QueryTx[sqlRow](tx), expression).List()
			return err
		}))
		names := make([]string, len(got))
		for i := range got {
			names[i] = got[i].Name
		}
		testutil.Equals(t, names, test.want)
	}
}

type sqlTimedRow struct {
	ID        int
	CreatedAt time.Time `store:"index"`
}
type sqlTimedView struct {
	CreatedAt time.Time `expr:"created_at" filter:"Created" filter-column:"CreatedAt"`
}

func TestApplySQLPushesCheckedDateLiteral(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s, err := store.Open(ctx, filepath.Join(t.TempDir(), "date.db"))
	testutil.Ok(t, err)
	t.Cleanup(func() { testutil.Ok(t, s.Close()) })
	s.Register(ctx, sqlTimedRow{})
	testutil.Ok(t, s.Write(ctx, func(tx *store.Tx) error {
		return tx.Insert(
			&sqlTimedRow{ID: 1, CreatedAt: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)},
			&sqlTimedRow{ID: 2, CreatedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
		)
	}))
	expression, err := filter.Parse(filter.NewSchema[sqlTimedView](), `created_at >= date("2026-07-01T00:00:00Z")`)
	testutil.Ok(t, err)
	var got []sqlTimedRow
	testutil.Ok(t, s.Read(ctx, func(tx *store.Tx) error {
		got, err = filter.ApplySQL(store.QueryTx[sqlTimedRow](tx), expression, func(r sqlTimedRow) sqlTimedView { return sqlTimedView{r.CreatedAt} }).List()
		return err
	}))
	testutil.ErrorIf(t, len(got) != 1 || got[0].ID != 2, "rows = %#v", got)
}

func pushdownNames(t *testing.T, rows []sqlRow, source string) []string {
	t.Helper()
	ctx, s := sqlStore(t, rows)
	expression, err := filter.Parse(filter.NewSchema[sqlView](), source)
	testutil.Ok(t, err)
	var got []sqlRow
	testutil.Ok(t, s.Read(ctx, func(tx *store.Tx) error {
		got, err = filter.ApplySQLPushdowns(store.QueryTx[sqlRow](tx), expression).List()
		return err
	}))
	names := make([]string, len(got))
	for i := range got {
		names[i] = got[i].Name
	}
	return names
}

func sqlStore(t *testing.T, rows []sqlRow) (context.Context, *store.Store) {
	t.Helper()
	ctx := context.Background()
	s, err := store.Open(ctx, filepath.Join(t.TempDir(), "filter.db"))
	testutil.Ok(t, err)
	t.Cleanup(func() { testutil.Ok(t, s.Close()) })
	s.Register(ctx, sqlRow{})
	testutil.Ok(t, s.Write(ctx, func(tx *store.Tx) error {
		for i := range rows {
			if err := tx.Insert(&rows[i]); err != nil {
				return err
			}
		}
		return nil
	}))
	return ctx, s
}
