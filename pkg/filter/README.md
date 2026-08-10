# Typed filter expressions

`pkg/filter` provides the transport-neutral expression language used by application list
operations. A domain declares a typed filter view, parsing checks user input against that view, and
the resulting expression can both evaluate complete values and narrow SQLite queries without
changing the expression's meaning.

For commands that exercise the language, see the
[filtering feature guide](../../docs/features.md#filtering-and-paging).

## Data path

```text
list request string -> Parse(domain schema) -> checked Expression
                                           -> safe SQLite pushdowns
                                           -> fetch and hydrate rows
                                           -> Match complete filter view
                                           -> authorization and paging
```

The schema is the public filter contract; it need not mirror a persistence row or a returned domain
model. This separation lets a list expose stable field names and derived or hydrated values such as
tags while keeping database details private.

## Declaring a schema

Schemas are derived from an exported struct. Tags define the external field name, help text, and,
where safe, the corresponding SQLite column:

```go
type RecipeFilterView struct {
	Garnish string `expr:"garnish" filter:"Recipe garnish"`
}

type DrinkFilterView struct {
	ID       string           `expr:"id" filter:"Drink ID" filter-column:"ID"`
	Name     string           `expr:"name" filter:"Drink name" filter-column:"Name"`
	Tags     []string         `expr:"tags" filter:"Tags (key or key=value)"`
	Recipe   RecipeFilterView `expr:"recipe"`
}

var drinkFilters = filter.NewSchema[DrinkFilterView](
	`name.contains("gin")`,
	`recipe.garnish.startsWith("lemon") || tags contains "featured"`,
)
```

- `expr` selects the user-visible name; without it, the Go field name is used. `expr:"-"` excludes
  a field.
- `filter` exposes the field and supplies its description to callers such as CLI filter help.
- `filter-column` names a field on the persisted SQLite row and enables safe pushdown. It is an
  optimization hint, not a rename of the filter field.
- Nested structs produce dotted paths such as `recipe.garnish`. `time.Time` is treated as one value
  rather than recursively exposed.
- `Schema.Fields` and `Schema.Examples` return copies suitable for presentation adapters.

Only exported fields with `filter` or `filter-column` metadata enter the schema. Use
`filter-column` only when the filter value and persisted column have compatible types and semantics;
hydrated fields such as `tags` must not claim a row column.

## Parsing and matching

```go
expression, err := filter.Parse(drinkFilters,
	`name.contains("gin") && tags contains "featured"`)
if err != nil {
	return err
}

matched, err := expression.Match(DrinkFilterView{
	Name: "London Dry Gin",
	Tags: []string{"featured"},
})
```

`Parse` trims the source and returns `nil, nil` for an empty expression. A nil expression matches
every value, so callers can pass it through the query and matching APIs without a separate empty
filter path. Invalid fields, incompatible types, unsupported constructs, invalid regular
expressions, and invalid time or duration literals return a typed invalid-input error before a
query runs.

An expression retains three useful representations:

- `Source` is the trimmed user input.
- `String` is canonical syntax suitable for display or reparsing.
- `Tree` is the package-owned `Node` tree for integrations that should not depend on Expr's AST.

### Supported syntax

- comparisons: `==`, `!=`, `<`, `<=`, `>`, `>=`, `in`, and `not in`;
- boolean logic: `&&`/`and`, `||`/`or`, `!`/`not`, and parentheses;
- string predicates: `contains`, `startsWith`, `endsWith`, and `matches`, in method or infix form;
- collection membership: `tags contains "featured"` or `tags.contains("featured")`;
- checked literals: `date("2026-07-01T00:00:00Z")` and `duration("2h30m")`.

`matches` requires a string-literal regular expression. Arbitrary function calls, arithmetic, and
other Expr constructs are intentionally rejected so accepted filters remain predictable and safe
to evaluate.

## Applying expressions to SQLite

Use `ApplySQL` when a complete filter view can be projected directly from one persisted row:

```go
q := store.QueryTx[AuditEntryRow](tx)
q = filter.ApplySQL(q, expression, func(row AuditEntryRow) AuditFilterView {
	return AuditFilterView{
		ID: row.ID, StartedAt: row.StartedAt, Success: row.Success,
	}
})
rows, err := q.List()
```

It adds persisted constraints that the optimizer can prove are required, then retains the complete
expression as an in-memory residual `FilterFn`. Comparisons, booleans, negation, and constraints common to
alternatives may be pushed down; string predicates and other residual logic still receive exact
in-memory evaluation.

Use the staged API when the view needs tags or other data loaded after the initial rows:

```go
q := filter.ApplySQLPushdowns(store.QueryTx[DrinkRow](tx), expression)
rows, err := q.List()
// Load tags for rows in one batch.
for _, row := range rows {
	view := DrinkFilterView{ID: row.ID, Name: row.Name, Tags: tagsByID[row.ID]}
	matched, err := expression.Match(view)
	// Yield only matched rows; propagate err.
}
```

`ApplySQLPushdowns` deliberately returns candidates, not final matches. Every candidate must be
projected with all derived data and passed to `Match`; omitting that step can return rows that do
not satisfy the user's expression. The drinks, ingredients, inventory, menus, and orders DAOs are
representative staged implementations, while audit demonstrates direct `ApplySQL` use.

## Extending a domain filter

1. Add the field and tags to the domain-owned filter view, keeping its external name stable.
2. Populate it wherever the complete view is projected; batch-load hydrated dependencies before
   calling `Match`.
3. Add a representative schema example so every presentation surface can teach the feature.
4. Add parse/match coverage and, for persisted fields, a test proving pushdown does not change the
   result set.

Changing the language itself requires updating validation, canonical formatting, the owned tree,
evaluation, and pushdown safety together. Start with exact evaluation; add a database optimization
only when it is logically implied by the full expression.
