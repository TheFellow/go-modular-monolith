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
├── keys.go             # Add Refresh key binding (r)
├── app.go              # Update to pass app to Dashboard, register domain ViewModels
├── views/
│   └── dashboard.go    # Update to accept app.App and show counts/activity
└── components/
    ├── detail_pane.go  # Shared DetailPane wrapper component
    ├── filter.go       # FilterDropdown component
    ├── search.go       # SearchInput component
    ├── spinner.go      # LoadingSpinner component
    ├── empty.go        # EmptyState component
    └── badge.go        # StatusBadge component
```

### Key Implementation Notes from Sprint 001

The following patterns were established in Sprint 001 and must be followed:

1. **App type**: Use `*app.App`, not `*app.Application`
2. **ViewModel interface**: Import from `github.com/TheFellow/go-modular-monolith/main/tui/views`
3. **Styles/Keys subset pattern**: Views receive only the styles/keys they need via dedicated struct types
   (e.g., `DashboardStyles`, `DashboardKeys`), not the full `Styles`/`KeyMap`. This provides better encapsulation.
4. **Message types**: Core messages (`NavigateMsg`, `ErrorMsg`, etc.) are in `main/tui/views/messages.go`
   with re-exports in `main/tui/messages.go` for convenience

### ViewModel Pattern

Each domain has two read-only ViewModels:

| ViewModel         | Purpose                                            | Saga-backed? |
|-------------------|----------------------------------------------------|--------------|
| `ListViewModel`   | Query and display list, handle filtering/selection | No           |
| `DetailViewModel` | Query and display single entity details            | No           |

These ViewModels implement the `ViewModel` interface from Sprint 001 and follow MVVM principles where the ViewModel
adapts domain data for the View layer.

## Tasks

### Phase 0: Infrastructure Updates

**Key Bindings:**
- [ ] Add `Refresh` key binding to `main/tui/keys.go` (`r` key, help text "refresh")
- [ ] Update `KeyMap.ShortHelp()` and `FullHelp()` to include Refresh where appropriate

**Styles/Keys for Domain ViewModels:**
- [ ] Create `ListViewStyles` and `ListViewKeys` subset types for domain ViewModels
- [ ] Update `App.currentViewModel()` to instantiate domain ViewModels (replace placeholders)

**TUI Error Surface (following existing error generation pattern):**
- [ ] Add `TUIStyle` type to `pkg/errors/errors.go` with values: `TUIStyleError`, `TUIStyleWarning`, `TUIStyleInfo`
- [ ] Add `TUIStyle TUIStyle` field to `ErrorKind` struct
- [ ] Assign appropriate `TUIStyle` to each error kind:
    - `Invalid` → `TUIStyleError` (user input error)
    - `NotFound` → `TUIStyleWarning` (informational)
    - `Permission` → `TUIStyleError` (access denied)
    - `Conflict` → `TUIStyleWarning` (recoverable)
    - `Internal` → `TUIStyleError` (unexpected)
- [ ] Update `pkg/errors/gen/errors.go.tpl` to generate `TUIStyle()` method
- [ ] Run `go generate ./pkg/errors/...` to regenerate error types
- [ ] Create `pkg/errors/tui.go` with:
    ```go
    // TUIError represents an error formatted for TUI display
    type TUIError struct {
        Style   TUIStyle
        Message string
        Err     error
    }

    // ToTUIError converts any error to a TUIError with appropriate styling
    func ToTUIError(err error) TUIError
    ```
- [ ] Update `main/tui/app.go` to use `ToTUIError()` when handling `ErrorMsg`
- [ ] Add `styles.WarningText` and `styles.InfoText` styles to complement `styles.ErrorText`

### Phase 1: Dashboard View Enhancement

- [ ] Update `Dashboard` to accept `*app.App` for data queries
- [ ] Create `DashboardData` struct to hold loaded counts/activity
- [ ] Implement async `loadDashboardData()` command
- [ ] Implement summary cards showing counts:
    - Total drinks
    - Total ingredients
    - Total menus (draft/published breakdown)
    - Low stock items count
    - Pending orders count
- [ ] Add recent activity feed (last 10 audit entries)
- [ ] Number keys (1-6) already navigate (from Sprint 001)
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
- [ ] All views handle terminal resize gracefully (inherited from Sprint 001)
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

## Implementation Details

### ListViewModel Pattern

Each domain's `ListViewModel` follows this structure. This example shows `drinks/surfaces/tui/list_vm.go`:

```go
// app/domains/drinks/surfaces/tui/list_vm.go
package tui

import (
    "context"

    "github.com/charmbracelet/bubbles/key"
    "github.com/charmbracelet/bubbles/list"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"

    "github.com/TheFellow/go-modular-monolith/app"
    "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
    "github.com/TheFellow/go-modular-monolith/main/tui/views"
)

// ListViewStyles contains styles needed by list views (subset of main Styles)
type ListViewStyles struct {
    Title      lipgloss.Style
    Muted      lipgloss.Style
    ListPane   lipgloss.Style
    DetailPane lipgloss.Style
    // ... other styles as needed
}

// ListViewKeys contains keys needed by list views (subset of main KeyMap)
type ListViewKeys struct {
    Up      key.Binding
    Down    key.Binding
    Enter   key.Binding
    Refresh key.Binding
    Back    key.Binding
}

// ListViewModel displays a filterable list of drinks with a detail pane
type ListViewModel struct {
    app      *app.App
    list     list.Model
    detail   *DetailViewModel  // Embedded detail view
    drinks   []models.Drink
    selected *models.Drink
    loading  bool
    err      error
    filter   string
    width    int
    height   int
    keys     ListViewKeys
    styles   ListViewStyles
}

func NewListViewModel(application *app.App, styles ListViewStyles, keys ListViewKeys) *ListViewModel {
    return &ListViewModel{
        app:    application,
        detail: NewDetailViewModel(styles),
        styles: styles,
        keys:   keys,
        // ... init list component
    }
}

func (vm *ListViewModel) Init() tea.Cmd {
    return vm.loadDrinks()
}

func (vm *ListViewModel) loadDrinks() tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background() // Or pass context through
        drinks, err := vm.app.Drinks.List(ctx)
        if err != nil {
            return views.ErrorMsg{Err: err}
        }
        return DrinksLoadedMsg{Drinks: drinks}
    }
}

func (vm *ListViewModel) Update(msg tea.Msg) (views.ViewModel, tea.Cmd) {
    switch msg := msg.(type) {
    case DrinksLoadedMsg:
        vm.drinks = msg.Drinks
        vm.loading = false
        vm.list.SetItems(toListItems(msg.Drinks))
    case tea.WindowSizeMsg:
        vm.width = msg.Width
        vm.height = msg.Height
    case tea.KeyMsg:
        if key.Matches(msg, vm.keys.Refresh) {
            vm.loading = true
            return vm, vm.loadDrinks()
        }
    }

    var cmd tea.Cmd
    vm.list, cmd = vm.list.Update(msg)

    // Update detail view when selection changes
    if i, ok := vm.list.SelectedItem().(drinkItem); ok {
        vm.selected = &i.drink
        vm.detail.SetDrink(&i.drink)
    }

    return vm, cmd
}

func (vm *ListViewModel) View() string {
    if vm.loading {
        return vm.styles.Muted.Render("Loading drinks...")
    }

    listView := vm.list.View()
    detailView := vm.detail.View()

    return lipgloss.JoinHorizontal(
        lipgloss.Top,
        vm.styles.ListPane.Render(listView),
        vm.styles.DetailPane.Render(detailView),
    )
}

func (vm *ListViewModel) ShortHelp() []key.Binding {
    return []key.Binding{vm.keys.Enter, vm.keys.Refresh, vm.keys.Back}
}

func (vm *ListViewModel) FullHelp() [][]key.Binding {
    return [][]key.Binding{
        {vm.keys.Up, vm.keys.Down, vm.keys.Enter},
        {vm.keys.Refresh, vm.keys.Back},
    }
}
```

### DetailViewModel Pattern

Each domain's `DetailViewModel` renders a single entity. This example shows `drinks/surfaces/tui/detail_vm.go`:

```go
// app/domains/drinks/surfaces/tui/detail_vm.go
package tui

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"

    "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
)

// DetailViewStyles contains styles needed by detail views
type DetailViewStyles struct {
    Title    lipgloss.Style
    Subtitle lipgloss.Style
    Label    lipgloss.Style
    Muted    lipgloss.Style
}

// DetailViewModel displays details for a single drink
// Note: This is embedded in ListViewModel, not a standalone ViewModel
type DetailViewModel struct {
    drink  *models.Drink
    styles DetailViewStyles
}

func NewDetailViewModel(styles ListViewStyles) *DetailViewModel {
    return &DetailViewModel{
        styles: DetailViewStyles{
            Title:    styles.Title,
            Subtitle: styles.Title, // Derive from parent styles
            Label:    styles.Muted.Bold(true),
            Muted:    styles.Muted,
        },
    }
}

func (vm *DetailViewModel) SetDrink(drink *models.Drink) {
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
    b.WriteString(d.Category.String())
    b.WriteString("\n")

    b.WriteString(vm.styles.Label.Render("Glass: "))
    b.WriteString(d.Glass.String())
    b.WriteString("\n")

    // Note: Check actual domain.Drink fields for price/ingredients structure

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

**Important**: The `DetailViewModel` shown here is a helper component, not a full `views.ViewModel`. It doesn't implement
`Init()`, `Update()`, `ShortHelp()`, or `FullHelp()` because it's managed by its parent `ListViewModel`.

### Async Command Pattern

Each domain defines its own loaded message type in its TUI package:

```go
// app/domains/drinks/surfaces/tui/messages.go
package tui

import "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"

// DrinksLoadedMsg is sent when drinks have been loaded from the app layer
type DrinksLoadedMsg struct {
    Drinks []models.Drink
}

// Similarly for other domains:
// app/domains/ingredients/surfaces/tui/messages.go -> IngredientsLoadedMsg
// app/domains/inventory/surfaces/tui/messages.go  -> InventoryLoadedMsg
// app/domains/menu/surfaces/tui/messages.go       -> MenusLoadedMsg
// app/domains/orders/surfaces/tui/messages.go     -> OrdersLoadedMsg
// app/domains/audit/surfaces/tui/messages.go      -> AuditLoadedMsg
```

The shared error message (`views.ErrorMsg`) is already defined in `main/tui/views/messages.go` from Sprint 001.

### TUI Error Surface Pattern

Following the existing error generation pattern in `pkg/errors/gen`, the TUI surface needs its own error styling.
The generator already produces surface-specific methods (`HTTPCode()`, `GRPCCode()`, `CLICode()`), and we extend it
to support TUI:

```go
// pkg/errors/errors.go - Add TUI style type and field

type TUIStyle int

const (
    TUIStyleError   TUIStyle = iota // Red - user errors, permission denied, internal errors
    TUIStyleWarning                  // Amber - not found, conflicts (recoverable)
    TUIStyleInfo                     // Muted - informational messages
)

type ErrorKind struct {
    Name     string
    Message  string
    HTTPCode httpCode
    GRPCCode codes.Code
    CLICode  int
    TUIStyle TUIStyle  // NEW: TUI presentation style
}

var ErrInvalid = ErrorKind{
    // ... existing fields
    TUIStyle: TUIStyleError,
}

var ErrNotFound = ErrorKind{
    // ... existing fields
    TUIStyle: TUIStyleWarning,  // "Not found" is often informational
}
```

```go
// pkg/errors/tui.go - TUI surface helpers

package errors

// TUIError represents an error formatted for TUI display
type TUIError struct {
    Style   TUIStyle
    Message string
    Err     error
}

// ToTUIError converts any error to a TUIError with appropriate styling
func ToTUIError(err error) TUIError {
    if err == nil {
        return TUIError{}
    }

    // Check for typed errors with TUIStyle() method
    type tuiStyler interface {
        TUIStyle() TUIStyle
    }

    style := TUIStyleError // Default to error style
    if ts, ok := err.(tuiStyler); ok {
        style = ts.TUIStyle()
    }

    return TUIError{
        Style:   style,
        Message: err.Error(),
        Err:     err,
    }
}
```

The `main/tui/app.go` uses this in the status bar:

```go
func (a *App) statusBarView() string {
    if a.lastError != nil {
        tuiErr := errors.ToTUIError(a.lastError)
        var style lipgloss.Style
        switch tuiErr.Style {
        case errors.TUIStyleError:
            style = a.styles.ErrorText
        case errors.TUIStyleWarning:
            style = a.styles.WarningText
        case errors.TUIStyleInfo:
            style = a.styles.InfoText
        }
        return a.styles.StatusBar.Render(style.Render(tuiErr.Message))
    }
    // ... normal status
}
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
