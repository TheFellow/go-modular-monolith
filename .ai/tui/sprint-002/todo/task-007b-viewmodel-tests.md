# Task 007b: ViewModel Tests (Intermezzo)

## Goal

Add black-box tests for the ViewModels already built (Drinks, Ingredients, Inventory) to establish guardrails around expected behavior.

## Design Principles

- **Black-box testing** - Test via public interface (Init, Update, View)
- **Use testutil.Fixture** - Real app with in-memory database, seeded data
- **Assert on View output** - Verify expected content appears
- **Verify errors surface** - Errors should appear in view, never swallowed

## Files to Create

- `app/domains/drinks/surfaces/tui/list_vm_test.go`
- `app/domains/ingredients/surfaces/tui/list_vm_test.go`
- `app/domains/inventory/surfaces/tui/list_vm_test.go`

## Test Pattern

```go
package tui_test

import (
    "strings"
    "testing"

    "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
    tui "github.com/TheFellow/go-modular-monolith/app/domains/drinks/surfaces/tui"
    "github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
    "github.com/TheFellow/go-modular-monolith/pkg/testutil"
    "github.com/charmbracelet/bubbles/key"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

func TestListViewModel_ShowsDrinksAfterLoad(t *testing.T) {
    t.Parallel()
    f := testutil.NewFixture(t)
    b := f.Bootstrap().WithBasicIngredients()

    lime := b.WithIngredient("Lime Juice", measurement.UnitOz)
    b.WithDrink(models.Drink{
        Name:     "Margarita",
        Category: models.DrinkCategoryCocktail,
        Recipe: models.Recipe{
            Ingredients: []models.RecipeIngredient{
                {IngredientID: lime.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)},
            },
            Steps: []string{"Shake"},
        },
    })

    vm := tui.NewListViewModel(f.App, f.OwnerContext(), testStyles(), testKeys())

    // Simulate Init and data load
    cmd := vm.Init()
    msg := runCmd(cmd)
    vm, _ = vm.Update(msg)
    vm, _ = vm.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

    view := vm.View()

    if !strings.Contains(view, "Margarita") {
        t.Errorf("expected view to contain 'Margarita', got:\n%s", view)
    }
}

func TestListViewModel_ShowsLoadingState(t *testing.T) {
    t.Parallel()
    f := testutil.NewFixture(t)

    vm := tui.NewListViewModel(f.App, f.OwnerContext(), testStyles(), testKeys())
    _ = vm.Init() // Don't process the command yet

    view := vm.View()

    if !strings.Contains(view, "Loading") {
        t.Errorf("expected loading state, got:\n%s", view)
    }
}

func TestListViewModel_ShowsEmptyState(t *testing.T) {
    t.Parallel()
    f := testutil.NewFixture(t)
    // No drinks seeded

    vm := tui.NewListViewModel(f.App, f.OwnerContext(), testStyles(), testKeys())
    cmd := vm.Init()
    msg := runCmd(cmd)
    vm, _ = vm.Update(msg)
    vm, _ = vm.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

    view := vm.View()

    // Should show empty list, not crash
    if view == "" {
        t.Error("expected non-empty view for empty list")
    }
}

// runCmd executes a tea.Cmd and returns the resulting message
func runCmd(cmd tea.Cmd) tea.Msg {
    if cmd == nil {
        return nil
    }
    return cmd()
}

// testStyles returns minimal styles for testing
func testStyles() tui.ListViewStyles {
    return tui.ListViewStyles{
        Title:       lipgloss.NewStyle(),
        Subtitle:    lipgloss.NewStyle(),
        Muted:       lipgloss.NewStyle(),
        Selected:    lipgloss.NewStyle(),
        ListPane:    lipgloss.NewStyle(),
        DetailPane:  lipgloss.NewStyle(),
        ErrorText:   lipgloss.NewStyle(),
        WarningText: lipgloss.NewStyle(),
    }
}

// testKeys returns key bindings for testing
func testKeys() tui.ListViewKeys {
    return tui.ListViewKeys{
        Up:      key.NewBinding(key.WithKeys("up")),
        Down:    key.NewBinding(key.WithKeys("down")),
        Enter:   key.NewBinding(key.WithKeys("enter")),
        Refresh: key.NewBinding(key.WithKeys("r")),
        Back:    key.NewBinding(key.WithKeys("esc")),
    }
}
```

## Test Scenarios Per ViewModel

### All ViewModels

| Test | Verifies |
|------|----------|
| `ShowsDataAfterLoad` | View contains expected entities after load completes |
| `ShowsLoadingState` | Loading spinner shown before data arrives |
| `ShowsEmptyState` | Empty list renders without error |
| `ShowsErrorOnFailure` | Errors displayed in view, not swallowed |

### Drinks-specific

| Test | Verifies |
|------|----------|
| `DetailShowsIngredients` | Selected drink shows ingredient names (not IDs) |
| `DetailShowsRecipeSteps` | Recipe steps displayed correctly |

### Ingredients-specific

| Test | Verifies |
|------|----------|
| `ShowsCategoryAndUnit` | Category and unit displayed in list |

### Inventory-specific

| Test | Verifies |
|------|----------|
| `ShowsStockStatus` | LOW/OUT status shown for low stock items |
| `ShowsIngredientName` | Ingredient name resolved (not just ID) |

## Checklist

- [ ] Create drinks list_vm_test.go with core tests
- [ ] Create ingredients list_vm_test.go with core tests
- [ ] Create inventory list_vm_test.go with core tests
- [ ] All tests pass with `go test ./app/domains/*/surfaces/tui/...`
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
