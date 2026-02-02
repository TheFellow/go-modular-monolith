# Task 007: Update Tests

## Goal

Update all test files to use the shared types from `pkg/tui` and verify all tests pass after the refactoring.

## Files to Modify

- `app/domains/drinks/surfaces/tui/list_vm_test.go`
- `app/domains/drinks/surfaces/tui/detail_vm_test.go`
- `app/domains/ingredients/surfaces/tui/list_vm_test.go`
- `app/domains/ingredients/surfaces/tui/detail_vm_test.go`
- `app/domains/inventory/surfaces/tui/list_vm_test.go`
- `app/domains/inventory/surfaces/tui/detail_vm_test.go`
- `app/domains/menus/surfaces/tui/list_vm_test.go`
- `app/domains/menus/surfaces/tui/detail_vm_test.go`
- `app/domains/orders/surfaces/tui/list_vm_test.go`
- `app/domains/orders/surfaces/tui/detail_vm_test.go`
- `app/domains/audit/surfaces/tui/list_vm_test.go`
- `app/domains/audit/surfaces/tui/detail_vm_test.go`

## Implementation

### 1. Update test helper imports:

```go
// Before
func testStyles() tui.ListViewStyles {
    return tui.ListViewStyles{...}
}

// After
import "github.com/TheFellow/go-modular-monolith/pkg/tui"

func testStyles() tui.ListViewStyles {
    return pkgtui.ListViewStyles{...}
}
```

### 2. Update test helper for keys:

```go
func testKeys() pkgtui.ListViewKeys {
    return pkgtui.ListViewKeys{
        Up:      key.NewBinding(key.WithKeys("up")),
        Down:    key.NewBinding(key.WithKeys("down")),
        Enter:   key.NewBinding(key.WithKeys("enter")),
        Refresh: key.NewBinding(key.WithKeys("r")),
        Back:    key.NewBinding(key.WithKeys("esc")),
    }
}
```

### 3. Update filter usage in tests:

```go
// Before
import drinksdao "github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
// drinksdao.ListFilter{}

// After
import drinksqueries "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
// drinksqueries.ListFilter{}
```

### 4. Add test for batch ingredient fetching:

```go
func TestDetailViewModel_BatchFetchesIngredients(t *testing.T) {
    t.Parallel()
    f := testutil.NewFixture(t)
    b := f.Bootstrap().WithBasicIngredients()

    // Create drink with multiple ingredients
    lime := b.WithIngredient("Lime Juice", measurement.UnitOz)
    tequila := b.WithIngredient("Tequila", measurement.UnitOz)
    salt := b.WithIngredient("Salt", measurement.UnitDash)

    drink := b.WithDrink(models.Drink{
        Name: "Margarita",
        Recipe: models.Recipe{
            Ingredients: []models.RecipeIngredient{
                {IngredientID: lime.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)},
                {IngredientID: tequila.ID, Amount: measurement.MustAmount(2, measurement.UnitOz)},
                {IngredientID: salt.ID, Amount: measurement.MustAmount(1, measurement.UnitDash)},
            },
        },
    })

    detail := tui.NewDetailViewModel(testStyles(), f.OwnerContext())
    detail.SetDrink(drink)

    view := detail.View()

    // All ingredient names should appear (from batch fetch)
    if !strings.Contains(view, "Lime Juice") {
        t.Error("expected Lime Juice in view")
    }
    if !strings.Contains(view, "Tequila") {
        t.Error("expected Tequila in view")
    }
    if !strings.Contains(view, "Salt") {
        t.Error("expected Salt in view")
    }
}
```

## Notes

- Run tests after each domain update to catch issues early
- Some test files may not exist yet (created in sprint-002)
- Focus on import changes and type usage

## Checklist

- [ ] Update drinks TUI test files
- [ ] Update ingredients TUI test files
- [ ] Update inventory TUI test files
- [ ] Update menus TUI test files
- [ ] Update orders TUI test files
- [ ] Update audit TUI test files
- [ ] Add batch ingredient fetch test
- [ ] `go test ./app/domains/*/surfaces/tui/...` passes
- [ ] `go test ./main/tui/...` passes
- [ ] `go test ./...` passes
