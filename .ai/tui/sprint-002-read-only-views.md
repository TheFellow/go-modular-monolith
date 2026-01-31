# Sprint 002: Read-Only Views

## Goal

Implement fully functional read-only views for all domains. Users can browse, search, filter, and view details for
drinks, ingredients, inventory, menus, orders, and audit logs. No create/update/delete operations yet.

## Problem

After Sprint 001, views are placeholders. Users cannot see any data from the application.

## Solution

Implement each view using Bubbles components (list, table, viewport) to display data from the `app` layer. Each list
view follows a consistent pattern: filterable list on the left, detail pane on the right.

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

- [ ] Create `DrinksModel` with bubbles/list component
- [ ] Implement `LoadDrinks()` command to fetch from `app.Drinks.List()`
- [ ] Configure list with:
    - Title: "Drinks"
    - Item delegate showing: name, category, glass, price
    - Filtering by typing
    - Status bar showing count
- [ ] Add detail pane (right side) showing selected drink:
    - Name, ID, category, glass, price
    - Ingredients list with quantities
- [ ] Implement filter dropdown for category (cocktail, shot, mocktail, beer)
- [ ] Add search input (fuzzy match on name)
- [ ] Handle empty state: "No drinks found"

### Phase 3: Ingredients View

- [ ] Create `IngredientsModel` with bubbles/list component
- [ ] Implement `LoadIngredients()` command
- [ ] Configure list with:
    - Title: "Ingredients"
    - Item delegate showing: name, category, unit
    - Filtering by typing
- [ ] Add detail pane showing:
    - Name, ID, category, unit
    - Current stock level (from inventory)
    - Drinks using this ingredient
- [ ] Implement filter dropdown for category (spirit, mixer, garnish, etc.)
- [ ] Handle empty state

### Phase 4: Inventory View

- [ ] Create `InventoryModel` with bubbles/table component
- [ ] Implement `LoadInventory()` command
- [ ] Configure table with columns:
    - Ingredient name
    - Category
    - Quantity (with unit)
    - Cost
    - Status (OK / LOW / OUT)
- [ ] Add "Show Low Stock Only" toggle (`!` key)
- [ ] Highlight low stock rows with warning color
- [ ] Add detail pane showing:
    - Full ingredient details
    - Stock history (recent adjustments from audit)
- [ ] Define low stock threshold (configurable or hardcoded initially)

### Phase 5: Menus View

- [ ] Create `MenusModel` with bubbles/list component
- [ ] Implement `LoadMenus()` command
- [ ] Configure list with:
    - Title: "Menus"
    - Item delegate showing: name, status (draft/published), drink count
    - Filtering by typing
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

- [ ] Create `OrdersModel` with bubbles/list component
- [ ] Implement `LoadOrders()` command
- [ ] Configure list with:
    - Title: "Orders"
    - Item delegate showing: ID (short), menu name, status, item count, total
    - Filtering by typing
- [ ] Add detail pane showing:
    - Full order ID, menu name, status
    - Line items (drink name × quantity, line total)
    - Order total
    - Timestamps (created, completed/cancelled if applicable)
- [ ] Implement filter dropdown for status (all, pending, completed, cancelled)
- [ ] Handle empty state

### Phase 7: Audit View

- [ ] Create `AuditModel` with bubbles/list or table component
- [ ] Implement `LoadAudit()` command with default limit (50 entries)
- [ ] Configure display showing:
    - Timestamp
    - Actor
    - Action (entity:operation)
    - Entity UID
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

- [ ] Create reusable `DetailPane` component for right-side details
- [ ] Create `FilterDropdown` component for category/status filters
- [ ] Create `SearchInput` component with debounced search
- [ ] Create `LoadingSpinner` component for async data fetches
- [ ] Create `EmptyState` component with customizable message
- [ ] Create `StatusBadge` component (draft/published, OK/LOW/OUT, etc.)

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

### List View Pattern

Each list view follows this structure:

```go
type DrinksModel struct {
    list     list.Model
    detail   DetailPane
    drinks   []domain.Drink
    selected *domain.Drink
    loading  bool
    err      error
    filter   string
    width    int
    height   int
    keys     DrinksKeyMap
}

func (m *DrinksModel) Init() tea.Cmd {
    return m.loadDrinks()
}

func (m *DrinksModel) loadDrinks() tea.Cmd {
    return func() tea.Msg {
        drinks, err := m.app.Drinks.List(ctx, queries.ListDrinksQuery{
            Category: m.filter,
        })
        if err != nil {
            return ErrorMsg{Err: err}
        }
        return DrinksLoadedMsg{Drinks: drinks}
    }
}

func (m *DrinksModel) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
    switch msg := msg.(type) {
    case DrinksLoadedMsg:
        m.drinks = msg.Drinks
        m.loading = false
        m.list.SetItems(toListItems(msg.Drinks))
    case list.Model:
        // Selection changed
        if i, ok := m.list.SelectedItem().(drinkItem); ok {
            m.selected = &i.drink
        }
    case tea.KeyMsg:
        if key.Matches(msg, m.keys.Refresh) {
            m.loading = true
            return m, m.loadDrinks()
        }
    }

    var cmd tea.Cmd
    m.list, cmd = m.list.Update(msg)
    return m, cmd
}

func (m *DrinksModel) View() string {
    if m.loading {
        return m.styles.Spinner.Render("Loading drinks...")
    }

    listView := m.list.View()
    detailView := m.renderDetail()

    return lipgloss.JoinHorizontal(
        lipgloss.Top,
        m.styles.ListPane.Render(listView),
        m.styles.DetailPane.Render(detailView),
    )
}
```

### Detail Pane Rendering

```go
func (m *DrinksModel) renderDetail() string {
    if m.selected == nil {
        return m.styles.Muted.Render("Select a drink to view details")
    }

    d := m.selected
    var b strings.Builder

    b.WriteString(m.styles.Title.Render(d.Name))
    b.WriteString("\n")
    b.WriteString(m.styles.Muted.Render(d.ID.String()))
    b.WriteString("\n\n")

    b.WriteString(m.styles.Label.Render("Category: "))
    b.WriteString(d.Category)
    b.WriteString("\n")

    b.WriteString(m.styles.Label.Render("Glass: "))
    b.WriteString(d.Glass)
    b.WriteString("\n")

    if d.Price != nil {
        b.WriteString(m.styles.Label.Render("Price: "))
        b.WriteString(fmt.Sprintf("$%.2f", *d.Price))
        b.WriteString("\n")
    }

    b.WriteString("\n")
    b.WriteString(m.styles.Subtitle.Render("Ingredients"))
    b.WriteString("\n")

    for _, ing := range d.Ingredients {
        b.WriteString(fmt.Sprintf("  • %s  %s\n", ing.Name, ing.Quantity))
    }

    return b.String()
}
```

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
