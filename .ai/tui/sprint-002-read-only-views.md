# Sprint 002: Read-Only Views

## Goal

Implement fully functional read-only views for all domains. Users can browse, search, filter, and view details for
drinks, ingredients, inventory, menus, orders, and audit logs. No create/update/delete operations yet.

## Problem

After Sprint 001, views are placeholders. Users cannot see any data from the application.

## Solution

Implement each view using Bubbles components (list, table, viewport) to display data from the `app` layer. Each list
view follows a consistent pattern: filterable list on the left, detail pane on the right.

This sprint establishes the **ListViewModel** and **DetailViewModel** patterns that will be reused and extended in
later sprints. These ViewModels are query-only (no sagas) and live under each domain's `surfaces/tui/` directory.

### Directory Structure

```
app/domains/drinks/surfaces/tui/
├── list_vm.go          # ListViewModel - list display with filtering
└── detail_vm.go        # DetailViewModel - single entity display

app/domains/ingredients/surfaces/tui/
├── list_vm.go
└── detail_vm.go

app/domains/inventory/surfaces/tui/
├── list_vm.go          # Uses table instead of list
└── detail_vm.go

app/domains/menus/surfaces/tui/
├── list_vm.go
└── detail_vm.go

app/domains/orders/surfaces/tui/
├── list_vm.go
└── detail_vm.go

app/domains/audit/surfaces/tui/
├── list_vm.go
└── detail_vm.go

main/tui/
├── views/
│   └── dashboard.go    # Dashboard is TUI-specific, not domain-owned
└── components/
    ├── detail_pane.go  # Shared DetailPane wrapper component
    ├── filter.go       # FilterDropdown component
    ├── search.go       # SearchInput component
    ├── spinner.go      # LoadingSpinner component
    ├── empty.go        # EmptyState component
    └── badge.go        # StatusBadge component
```

### ViewModel Pattern

Each domain has two read-only ViewModels:

| ViewModel         | Purpose                                            | Saga-backed? |
|-------------------|----------------------------------------------------|--------------|
| `ListViewModel`   | Query and display list, handle filtering/selection | No           |
| `DetailViewModel` | Query and display single entity details            | No           |

These ViewModels implement the `ViewModel` interface from Sprint 001 and follow MVVM principles where the ViewModel
adapts domain data for the View layer.

## Tasks

### Phase 1: Dashboard View

- [ ] Implement summary cards showing counts:
    - Total drinks
    - Total ingredients
    - Total menus (draft/published breakdown)
    - Low stock items count
    - Pending orders count
- [ ] Add recent activity feed (last 10 audit entries)
- [ ] Wire number keys (1-6) to navigate to respective views
- [ ] Add quick stats row (e.g., "12 drinks on active menus")

### Phase 2: Drinks View

- [ ] Create `app/domains/drinks/surfaces/tui/list_vm.go` with `ListViewModel`
- [ ] Implement `LoadDrinks()` command to fetch from `app.Drinks.List()`
- [ ] Configure list with:
    - Title: "Drinks"
    - Item delegate showing: name, category, glass, price
    - Filtering by typing
    - Status bar showing count
- [ ] Create `app/domains/drinks/surfaces/tui/detail_vm.go` with `DetailViewModel`
- [ ] Add detail pane (right side) showing selected drink:
    - Name, ID, category, glass, price
    - Ingredients list with quantities
- [ ] Implement filter dropdown for category (cocktail, shot, mocktail, beer)
- [ ] Add search input (fuzzy match on name)
- [ ] Handle empty state: "No drinks found"

### Phase 3: Ingredients View

- [ ] Create `app/domains/ingredients/surfaces/tui/list_vm.go` with `ListViewModel`
- [ ] Implement `LoadIngredients()` command
- [ ] Configure list with:
    - Title: "Ingredients"
    - Item delegate showing: name, category, unit
    - Filtering by typing
- [ ] Create `app/domains/ingredients/surfaces/tui/detail_vm.go` with `DetailViewModel`
- [ ] Add detail pane showing:
    - Name, ID, category, unit
    - Current stock level (from inventory)
    - Drinks using this ingredient
- [ ] Implement filter dropdown for category (spirit, mixer, garnish, etc.)
- [ ] Handle empty state

### Phase 4: Inventory View

- [ ] Create `app/domains/inventory/surfaces/tui/list_vm.go` with `ListViewModel` (uses table component)
- [ ] Implement `LoadInventory()` command
- [ ] Configure table with columns:
    - Ingredient name
    - Category
    - Quantity (with unit)
    - Cost
    - Status (OK / LOW / OUT)
- [ ] Add "Show Low Stock Only" toggle (`!` key)
- [ ] Highlight low stock rows with warning color
- [ ] Create `app/domains/inventory/surfaces/tui/detail_vm.go` with `DetailViewModel`
- [ ] Add detail pane showing:
    - Full ingredient details
    - Stock history (recent adjustments from audit)
- [ ] Define low stock threshold (configurable or hardcoded initially)

### Phase 5: Menus View

- [ ] Create `app/domains/menus/surfaces/tui/list_vm.go` with `ListViewModel`
- [ ] Implement `LoadMenus()` command
- [ ] Configure list with:
    - Title: "Menus"
    - Item delegate showing: name, status (draft/published), drink count
    - Filtering by typing
- [ ] Create `app/domains/menus/surfaces/tui/detail_vm.go` with `DetailViewModel`
- [ ] Add detail pane showing:
    - Name, ID, status
    - List of drinks on menu with prices
    - Cost analysis (if `--costs` equivalent):
        - Average cost
        - Average margin
        - Suggested prices
- [ ] Implement filter dropdown for status (all, draft, published)
- [ ] Handle empty state

### Phase 6: Orders View

- [ ] Create `app/domains/orders/surfaces/tui/list_vm.go` with `ListViewModel`
- [ ] Implement `LoadOrders()` command
- [ ] Configure list with:
    - Title: "Orders"
    - Item delegate showing: ID (short), menu name, status, item count, total
    - Filtering by typing
- [ ] Create `app/domains/orders/surfaces/tui/detail_vm.go` with `DetailViewModel`
- [ ] Add detail pane showing:
    - Full order ID, menu name, status
    - Line items (drink name × quantity, line total)
    - Order total
    - Timestamps (created, completed/cancelled if applicable)
- [ ] Implement filter dropdown for status (all, pending, completed, cancelled)
- [ ] Handle empty state

### Phase 7: Audit View

- [ ] Create `app/domains/audit/surfaces/tui/list_vm.go` with `ListViewModel`
- [ ] Implement `LoadAudit()` command with default limit (50 entries)
- [ ] Configure display showing:
    - Timestamp
    - Actor
    - Action (entity:operation)
    - Entity UID
- [ ] Create `app/domains/audit/surfaces/tui/detail_vm.go` with `DetailViewModel`
- [ ] Add detail pane showing:
    - Full audit entry details
    - Touched entities list
    - Before/after state (if available)
- [ ] Implement filters:
    - Entity type (Drink, Ingredient, Menu, etc.)
    - Action (create, update, delete)
    - Actor (owner, manager, etc.)
    - Time range (today, this week, all)
- [ ] Add "Jump to Entity" action (`enter` on audit entry navigates to entity)

### Phase 8: Shared Components

Create reusable components under `main/tui/components/`:

- [ ] Create `main/tui/components/detail_pane.go` - reusable detail pane wrapper
- [ ] Create `main/tui/components/filter.go` - `FilterDropdown` for category/status filters
- [ ] Create `main/tui/components/search.go` - `SearchInput` with debounced search
- [ ] Create `main/tui/components/spinner.go` - `LoadingSpinner` for async data fetches
- [ ] Create `main/tui/components/empty.go` - `EmptyState` with customizable message
- [ ] Create `main/tui/components/badge.go` - `StatusBadge` (draft/published, OK/LOW/OUT, etc.)

### Phase 9: Data Loading Pattern

- [ ] Implement async data loading with tea.Cmd
- [ ] Show spinner while loading
- [ ] Handle errors gracefully (show error in status bar, allow retry)
- [ ] Cache loaded data in view model
- [ ] Add manual refresh (`r` key)

## Acceptance Criteria

- [ ] Dashboard shows accurate counts and recent activity
- [ ] All list views display data from the app layer
- [ ] Filtering works (by typing in list, by dropdown for categories)
- [ ] Search works (fuzzy match on names)
- [ ] Detail pane updates when selection changes
- [ ] Empty states display appropriate messages
- [ ] Loading states show spinner
- [ ] Errors display in status bar with retry option
- [ ] `r` refreshes current view data
- [ ] All views handle terminal resize gracefully

## Implementation Details

### ListViewModel Pattern

Each domain's `ListViewModel` follows this structure. This example shows `drinks/surfaces/tui/list_vm.go`:

```go
// app/domains/drinks/surfaces/tui/list_vm.go
package tui

// ListViewModel displays a filterable list of drinks with a detail pane
type ListViewModel struct {
    app      *app.Application
    list     list.Model
    detail   *DetailViewModel  // Embedded detail view
    drinks   []domain.Drink
    selected *domain.Drink
    loading  bool
    err      error
    filter   string
    width    int
    height   int
    keys     KeyMap
    styles   Styles
}

func NewListViewModel(app *app.Application) *ListViewModel {
    return &ListViewModel{
        app:    app,
        detail: NewDetailViewModel(app),
        // ... init list component
    }
}

func (vm *ListViewModel) Init() tea.Cmd {
    return vm.loadDrinks()
}

func (vm *ListViewModel) loadDrinks() tea.Cmd {
    return func() tea.Msg {
        drinks, err := vm.app.Drinks.List(ctx, queries.ListDrinksQuery{
            Category: vm.filter,
        })
        if err != nil {
            return ErrorMsg{Err: err}
        }
        return DrinksLoadedMsg{Drinks: drinks}
    }
}

func (vm *ListViewModel) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
    switch msg := msg.(type) {
    case DrinksLoadedMsg:
        vm.drinks = msg.Drinks
        vm.loading = false
        vm.list.SetItems(toListItems(msg.Drinks))
    case list.Model:
        // Selection changed - update detail view
        if i, ok := vm.list.SelectedItem().(drinkItem); ok {
            vm.selected = &i.drink
            vm.detail.SetDrink(&i.drink)
        }
    case tea.KeyMsg:
        if key.Matches(msg, vm.keys.Refresh) {
            vm.loading = true
            return vm, vm.loadDrinks()
        }
    }

    var cmd tea.Cmd
    vm.list, cmd = vm.list.Update(msg)
    return vm, cmd
}

func (vm *ListViewModel) View() string {
    if vm.loading {
        return vm.styles.Spinner.Render("Loading drinks...")
    }

    listView := vm.list.View()
    detailView := vm.detail.View()

    return lipgloss.JoinHorizontal(
        lipgloss.Top,
        vm.styles.ListPane.Render(listView),
        vm.styles.DetailPane.Render(detailView),
    )
}
```

### DetailViewModel Pattern

Each domain's `DetailViewModel` renders a single entity. This example shows `drinks/surfaces/tui/detail_vm.go`:

```go
// app/domains/drinks/surfaces/tui/detail_vm.go
package tui

// DetailViewModel displays details for a single drink
type DetailViewModel struct {
    app    *app.Application
    drink  *domain.Drink
    styles Styles
}

func NewDetailViewModel(app *app.Application) *DetailViewModel {
    return &DetailViewModel{app: app}
}

func (vm *DetailViewModel) SetDrink(drink *domain.Drink) {
    vm.drink = drink
}

func (vm *DetailViewModel) View() string {
    if vm.drink == nil {
        return vm.styles.Muted.Render("Select a drink to view details")
    }

    d := vm.drink
    var b strings.Builder

    b.WriteString(vm.styles.Title.Render(d.Name))
    b.WriteString("\n")
    b.WriteString(vm.styles.Muted.Render(d.ID.String()))
    b.WriteString("\n\n")

    b.WriteString(vm.styles.Label.Render("Category: "))
    b.WriteString(d.Category)
    b.WriteString("\n")

    b.WriteString(vm.styles.Label.Render("Glass: "))
    b.WriteString(d.Glass)
    b.WriteString("\n")

    if d.Price != nil {
        b.WriteString(vm.styles.Label.Render("Price: "))
        b.WriteString(fmt.Sprintf("$%.2f", *d.Price))
        b.WriteString("\n")
    }

    b.WriteString("\n")
    b.WriteString(vm.styles.Subtitle.Render("Ingredients"))
    b.WriteString("\n")

    for _, ing := range d.Ingredients {
        b.WriteString(fmt.Sprintf("  • %s  %s\n", ing.Name, ing.Quantity))
    }

    return b.String()
}
```

The `DetailViewModel` is typically embedded within the `ListViewModel` and updated when selection changes. For full-screen
detail views (on narrow terminals), the same `DetailViewModel` can be used standalone.

### Async Command Pattern

```go
// Messages for async operations
type DrinksLoadedMsg struct{ Drinks []domain.Drink }
type IngredientsLoadedMsg struct{ Ingredients []domain.Ingredient }
type InventoryLoadedMsg struct{ Stock []domain.Stock }
type MenusLoadedMsg struct{ Menus []domain.Menu }
type OrdersLoadedMsg struct{ Orders []domain.Order }
type AuditLoadedMsg struct{ Entries []domain.AuditEntry }

// Generic error and loading messages
type ErrorMsg struct{ Err error }
type LoadingMsg struct{ View View }
```

## Notes

### Split Pane Layout

Views use a 60/40 split between list and detail pane. On narrow terminals (<100 columns), detail pane hides and `enter`
opens a full-screen detail view.

### Filter Persistence

Filter selections persist within a session. Navigating away and back remembers the filter state.

### ID Display

IDs are shown in the detail pane but truncated in list views (show last 8 characters). Full ID is copyable via `y` key
in detail view.

### Performance

Initial load fetches all items. For large datasets (>1000 items), implement pagination in future sprint. For now,
assume reasonable data sizes.

### MVVM Alignment

This sprint establishes the ViewModel pattern that aligns with MVVM principles:

| MVVM          | Sprint 002 Pattern | Purpose                                    |
|---------------|--------------------|--------------------------------------------|
| **Model**     | Domain entities    | Data from app layer queries                |
| **ViewModel** | ListViewModel      | Adapts list data for View, handles filters |
| **ViewModel** | DetailViewModel    | Adapts single entity for View              |
| **View**      | `View()` output    | Bubble Tea rendering                       |

The read-only ViewModels introduced here are **query-only**—they fetch and display data but don't modify it.
Sprint 004 introduces **CreateViewModel** which is saga-backed and handles mutations.

```
┌─────────────────────────────────────────────────────────────────────────┐
│  ViewModel Types by Sprint                                              │
├─────────────────────────────────────────────────────────────────────────┤
│  Sprint 002 (Read-Only):                                                │
│    ListViewModel    - Query list, filter, select                        │
│    DetailViewModel  - Query and display single entity                   │
│                                                                         │
│  Sprint 004 (Saga-Powered):                                             │
│    CreateViewModel  - Saga-backed creation workflows                    │
│    EditViewModel    - Saga-backed editing (often extends CreateViewModel)│
└─────────────────────────────────────────────────────────────────────────┘
```

### Why Domain-Owned ViewModels?

ViewModels live under `app/domains/*/surfaces/tui/` because:

1. **Domain cohesion**: Each domain owns its presentation logic
2. **Reusability**: ViewModels can be composed across different TUI contexts
3. **Testing**: Domain ViewModels can be tested with domain-specific mocks
4. **Consistency**: Mirrors the existing surfaces pattern (CLI surface exists similarly)
