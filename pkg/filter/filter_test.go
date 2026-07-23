package filter_test

import (
	"strings"
	"testing"
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/filter"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

type nested struct {
	Name string `expr:"name" filter:"Nested name"`
}
type view struct {
	Name     string `expr:"name" filter:"Display name" filter-column:"Name"`
	Category string `expr:"category" filter:"Category" filter-column:"Category"`
	Deleted  bool   `expr:"deleted" filter:"Whether deleted"`
	Active   bool   `expr:"active" filter:"Whether active"`
	Nested   nested `expr:"nested"`
}

func TestCanonicalPrecedenceAndBooleanSpellings(t *testing.T) {
	t.Parallel()

	schema := filter.NewSchema[view]()
	tests := map[string]string{
		`deleted or active and name == "x"`:  `deleted || active && name == "x"`,
		`(deleted || active) && name == "x"`: `(deleted || active) && name == "x"`,
		`not deleted or !active`:             `!deleted || !active`,
	}
	for source, want := range tests {
		expression, err := filter.Parse(schema, source)
		testutil.ErrorIf(t, err != nil, "Parse(%q): %v", source, err)
		testutil.Equals(t, expression.String(), want)
	}
}

func TestDotAndInfixStringPredicatesAreEquivalent(t *testing.T) {
	t.Parallel()

	schema := filter.NewSchema[view]()
	for _, predicate := range []string{"contains", "startsWith", "endsWith", "matches"} {
		dot, err := filter.Parse(schema, `name.`+predicate+`("gin")`)
		testutil.Ok(t, err)
		infix, err := filter.Parse(schema, `name `+predicate+` "gin"`)
		testutil.Ok(t, err)
		testutil.Equals(t, dot.String(), infix.String())
	}
}

func TestRejectsRuntimeFailableLiterals(t *testing.T) {
	t.Parallel()

	type timed struct {
		At time.Time `expr:"at" filter:"Time"`
	}
	for _, source := range []string{
		`at >= date("not-a-date")`,
		`name.matches("[")`,
		`name.matches(nested.name)`,
	} {
		var err error
		if strings.HasPrefix(source, "at") {
			_, err = filter.Parse(filter.NewSchema[timed](), source)
		} else {
			_, err = filter.Parse(filter.NewSchema[view](), source)
		}
		testutil.ErrorIf(t, err == nil, "Parse(%q) succeeded", source)
	}
}

func TestParseAliasesDotSyntaxAndRoundTrip(t *testing.T) {
	t.Parallel()

	schema := filter.NewSchema[view](`category == "spirit" && name.contains("gin")`)
	expression, err := filter.Parse(schema, `category == "spirit" and (name contains "gin" or not deleted)`)
	testutil.Ok(t, err)
	const want = `category == "spirit" && (name.contains("gin") || !deleted)`
	testutil.Equals(t, expression.String(), want)
	again, err := filter.Parse(schema, expression.String())
	testutil.Ok(t, err)
	testutil.Equals(t, again.String(), want)
	matched, err := expression.Match(view{Category: "spirit", Name: "London gin"})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, !matched, "filter did not match")
}

func TestNestedFieldAndBooleanSymbols(t *testing.T) {
	t.Parallel()

	expression, err := filter.Parse(filter.NewSchema[view](), `nested.name.startsWith("old") && !deleted`)
	testutil.Ok(t, err)
	matched, err := expression.Match(view{Nested: nested{Name: "old fashioned"}})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, !matched, "filter did not match")
}

func TestRejectsUnknownAndNonFilterConstructs(t *testing.T) {
	t.Parallel()

	for _, source := range []string{`missing == "x"`, `len(name) > 2`, `1 + 1 == 2`} {
		_, err := filter.Parse(filter.NewSchema[view](), source)
		testutil.ErrorIf(t, err == nil, "Parse(%q) succeeded", source)
	}
}
