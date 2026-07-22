package filter_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/filter"
)

type nested struct {
	Name string `expr:"name" filter:"Nested name"`
}
type view struct {
	Name     string `expr:"name" filter:"Display name" filter-column:"Name"`
	Category string `expr:"category" filter:"Category" filter-column:"Category"`
	Deleted  bool   `expr:"deleted" filter:"Whether deleted"`
	Nested   nested `expr:"nested"`
}

func TestParseAliasesDotSyntaxAndRoundTrip(t *testing.T) {
	schema := filter.NewSchema[view](`category == "spirit" && name.contains("gin")`)
	expression, err := filter.Parse(schema, `category == "spirit" and (name contains "gin" or not deleted)`)
	if err != nil {
		t.Fatal(err)
	}
	const want = `category == "spirit" && (name.contains("gin") || !deleted)`
	if expression.String() != want {
		t.Fatalf("canonical = %q, want %q", expression.String(), want)
	}
	again, err := filter.Parse(schema, expression.String())
	if err != nil {
		t.Fatal(err)
	}
	if again.String() != want {
		t.Fatalf("round trip = %q", again.String())
	}
	matched, err := expression.Match(view{Category: "spirit", Name: "London gin"})
	if err != nil || !matched {
		t.Fatalf("matched=%v err=%v", matched, err)
	}
}

func TestNestedFieldAndBooleanSymbols(t *testing.T) {
	expression, err := filter.Parse(filter.NewSchema[view](), `nested.name.startsWith("old") && !deleted`)
	if err != nil {
		t.Fatal(err)
	}
	matched, err := expression.Match(view{Nested: nested{Name: "old fashioned"}})
	if err != nil || !matched {
		t.Fatalf("matched=%v err=%v", matched, err)
	}
}

func TestRejectsUnknownAndNonFilterConstructs(t *testing.T) {
	for _, source := range []string{`missing == "x"`, `len(name) > 2`, `1 + 1 == 2`} {
		if _, err := filter.Parse(filter.NewSchema[view](), source); err == nil {
			t.Fatalf("Parse(%q) succeeded", source)
		}
	}
}
