# Proposal 006: Interactive TUI with Bubble Tea

**Status:** Proposed
**Type:** New Surface / User Experience Enhancement

## Goal

Add an interactive Terminal User Interface (TUI) to `mixology` via a `--tui` flag, using
the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework. The TUI provides all CLI capabilities in a more
user-friendly, navigable interface where users select items from lists rather than copying/pasting cryptic IDs.

## Motivation

### Current CLI Pain Points

1. **ID Juggling**: Users must copy/paste KSUIDs like `drk-2eQmj4GcwFLkqJ6hYZ8xk6Wk7vE` between commands
2. **Context Loss**: Each command is stateless; users lose context between operations
3. **Discovery**: New users must read help text to discover available operations
4. **Multi-Step Workflows**: Common flows (create drink → add to menu → publish) require multiple discrete commands

### TUI Benefits

1. **Selection-Based Navigation**: Choose drinks, ingredients, menus from lists—no ID memorization
2. **Contextual Actions**: See available operations for the currently selected entity
3. **Persistent State**: Browse while maintaining context (selected menu, active filters)
4. **Guided Workflows**: Multi-step forms with validation and confirmation
5. **Visual Feedback**: Real-time status, inline editing, hierarchical views

## Architecture Overview

### Entry Point

```
mixology --tui                    # Launch full TUI
mixology --tui drinks             # Launch TUI focused on drinks view
mixology --tui menu --id mnu-123  # Launch TUI with specific menu selected
```

The TUI surface lives alongside the CLI surface, sharing the same `app` layer.

```
main/tui/           # New TUI entry point
├── main.go         # Program initialization
├── app.go          # Root Bubble Tea model
├── styles.go       # Lip Gloss theme definitions
├── keys.go         # Key bindings
└── views/          # View-specific models
    ├── dashboard.go
    ├── drinks.go
    ├── ingredients.go
    ├── inventory.go
    ├── menus.go
    ├── orders.go
    └── audit.go
```

### Bubble Tea Components

| Component  | Library           | Purpose                                       |
|------------|-------------------|-----------------------------------------------|
| List       | bubbles/list      | Entity selection (drinks, ingredients, menus) |
| Table      | bubbles/table     | Tabular data display with sorting             |
| Text Input | bubbles/textinput | Single-field entry (name, price)              |
| Text Area  | bubbles/textarea  | Multi-line input (descriptions)               |
| Viewport   | bubbles/viewport  | Scrollable content areas                      |
| Spinner    | bubbles/spinner   | Loading states                                |
| Help       | bubbles/help      | Context-sensitive key hints                   |
| Lip Gloss  | lipgloss          | Styling, borders, colors                      |

## Screen Designs

### 1. Dashboard (Home)

```
┌─────────────────────────────────────────────────────────────────────┐
│  MIXOLOGY                                              [?] Help     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │
│   │  DRINKS     │  │ INGREDIENTS │  │   MENUS     │                 │
│   │     42      │  │     156     │  │      3      │                 │
│   └─────────────┘  └─────────────┘  └─────────────┘                 │
│                                                                     │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │
│   │  INVENTORY  │  │   ORDERS    │  │    AUDIT    │                 │
│   │  12 low     │  │  5 pending  │  │    today    │                 │
│   └─────────────┘  └─────────────┘  └─────────────┘                 │
│                                                                     │
│   Recent Activity:                                                  │
│   • Menu "Summer Specials" published                    2 min ago   │
│   • Inventory adjusted: Vodka +10 units                 5 min ago   │
│   • New drink "Sunset Spritz" created                  12 min ago   │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│ [1-6] Navigate  [/] Search  [c] Create  [q] Quit                    │
└─────────────────────────────────────────────────────────────────────┘
```

### 2. Drinks List View

```
┌─────────────────────────────────────────────────────────────────────┐
│  DRINKS                                    Filter: cocktail ▼   [x] │
├─────────────────────────────────────────────────────────────────────┤
│  Search: _                                                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  > Margarita                    cocktail      coupe         $12.00  │
│    Mojito                       cocktail      highball      $11.00  │
│    Old Fashioned                cocktail      rocks         $14.00  │
│    Whiskey Sour                 cocktail      coupe         $13.00  │
│    Mai Tai                      cocktail      tiki          $15.00  │
│    Daiquiri                     cocktail      coupe         $11.00  │
│    ...                                                              │
│                                                                     │
│  ───────────────────────────────────────────────────────────────    │
│  Margarita                                             drk-abc123   │
│  Category: cocktail | Glass: coupe | Price: $12.00                  │
│                                                                     │
│  Ingredients:                                                       │
│    • Tequila Blanco      2 oz                                       │
│    • Lime Juice          1 oz                                       │
│    • Triple Sec          0.75 oz                                    │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│ [↑↓] Navigate  [enter] Edit  [c] Create  [d] Delete  [m] Add to Menu│
│ [/] Search     [f] Filter    [esc] Back                             │
└─────────────────────────────────────────────────────────────────────┘
```

### 3. Create/Edit Form (Drinks)

```
┌─────────────────────────────────────────────────────────────────────┐
│  CREATE DRINK                                                   [x] │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Name:        [Sunset Spritz________________]                       │
│                                                                     │
│  Category:    [cocktail     ▼]                                      │
│               • cocktail                                            │
│               • shot                                                │
│               • mocktail                                            │
│               • beer                                                │
│                                                                     │
│  Glass:       [highball     ▼]                                      │
│                                                                     │
│  Price:       [$____________] (optional)                            │
│                                                                     │
│  ─── Ingredients ───────────────────────────────────────────────    │
│                                                                     │
│  [+] Add Ingredient                                                 │
│                                                                     │
│    1. Aperol                    1.5 oz          [edit] [x]          │
│    2. Prosecco                  3 oz            [edit] [x]          │
│    3. Soda Water                1 oz            [edit] [x]          │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│ [tab] Next Field  [shift+tab] Previous  [ctrl+s] Save  [esc] Cancel │
└─────────────────────────────────────────────────────────────────────┘
```

### 4. Menu Builder

```
┌─────────────────────────────────────────────────────────────────────┐
│  MENU: Summer Specials                           Status: draft  [x] │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─ Available Drinks ─────────────┐  ┌─ On This Menu ─────────────┐ │
│  │                                │  │                            │ │
│  │  > Margarita          $12.00   │  │    Mojito          $11.00  │ │
│  │    Old Fashioned      $14.00   │  │    Mai Tai         $15.00  │ │
│  │    Whiskey Sour       $13.00   │  │    Piña Colada     $13.00  │ │
│  │    Daiquiri           $11.00   │  │                            │ │
│  │    Negroni            $13.00   │  │                            │ │
│  │    Manhattan          $14.00   │  │                            │ │
│  │                                │  │                            │ │
│  │  [enter] Add to Menu ─────────►│  │◄───────── [del] Remove     │ │
│  │                                │  │                            │ │
│  └────────────────────────────────┘  └────────────────────────────┘ │
│                                                                     │
│  ─── Cost Analysis ─────────────────────────────────────────────    │
│  Total Items: 3    Avg Cost: $4.20    Avg Margin: 68%               │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│ [tab] Switch Panel  [p] Publish Menu  [r] Rename  [esc] Back        │
└─────────────────────────────────────────────────────────────────────┘
```

### 5. Order Placement

```
┌─────────────────────────────────────────────────────────────────────┐
│  PLACE ORDER                          Menu: Summer Specials     [x] │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─ Menu Items ───────────────────┐  ┌─ Order ────────────────────┐ │
│  │                                │  │                            │ │
│  │  > Mojito             $11.00   │  │  2x Mojito         $22.00  │ │
│  │    Mai Tai            $15.00   │  │  1x Piña Colada    $13.00  │ │
│  │    Piña Colada        $13.00   │  │                            │ │
│  │                                │  │                            │ │
│  │                                │  │  ────────────────────────  │ │
│  │                                │  │  Total:            $35.00  │ │
│  │                                │  │                            │ │
│  │  [enter] Add  [+/-] Quantity   │  │  [del] Remove Item         │ │
│  └────────────────────────────────┘  └────────────────────────────┘ │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│ [tab] Switch Panel  [ctrl+enter] Submit Order  [esc] Cancel         │
└─────────────────────────────────────────────────────────────────────┘
```

### 6. Inventory Management

```
┌─────────────────────────────────────────────────────────────────────┐
│  INVENTORY                                    [!] Show Low Stock    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Ingredient              Category    Qty      Cost     Status       │
│  ─────────────────────────────────────────────────────────────────  │
│  > Vodka                 spirit      8.5 L    $180     ⚠ LOW        │
│    Gin                   spirit      12 L     $240     OK           │
│    Tequila Blanco        spirit      6 L      $150     ⚠ LOW        │
│    Rum (White)           spirit      15 L     $225     OK           │
│    Lime Juice            mixer       3 L      $15      ⚠ LOW        │
│    Simple Syrup          mixer       5 L      $10      OK           │
│                                                                     │
│  ─── Adjust Stock ──────────────────────────────────────────────    │
│  Selected: Vodka                                                    │
│                                                                     │
│  Adjustment: [+5.0_____] L                                          │
│  Reason:     [received  ▼]  • received • used • spilled • expired   │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│ [↑↓] Navigate  [a] Adjust  [s] Set Exact  [enter] Apply  [esc] Back │
└─────────────────────────────────────────────────────────────────────┘
```

## Technical Strategy

### Phase 1: Foundation (Scaffolding)

1. Add Bubble Tea dependencies:
   ```
   github.com/charmbracelet/bubbletea
   github.com/charmbracelet/bubbles
   github.com/charmbracelet/lipgloss
   ```

2. Create `main/tui/` entry point with root model
3. Implement basic navigation between empty views
4. Define shared styles (colors, borders, spacing)
5. Implement key binding system and help overlay

### Phase 2: Read-Only Views

1. **Dashboard**: Summary cards, recent activity feed
2. **List Views**: Drinks, Ingredients, Menus, Orders with:
    - Filtering and searching
    - Detail pane for selected item
    - Pagination/scrolling
3. **Audit View**: Filterable log with entity links

### Phase 3: CRUD Operations

1. **Create Forms**: Multi-field forms with validation
2. **Edit Forms**: Pre-populated forms, dirty tracking
3. **Delete Confirmation**: Modal dialogs
4. **Inventory Adjustments**: Inline quantity editing

### Phase 4: Workflows

1. **Drink Builder**: Ingredient selection from list (no ID typing)
2. **Menu Builder**: Dual-pane add/remove interface
3. **Order Placement**: Menu-aware drink selection
4. **Quick Actions**: Keyboard shortcuts for common flows

### Phase 5: Polish

1. **Responsive Layout**: Adapt to terminal size
2. **Mouse Support**: Optional clickable elements
3. **Themes**: Light/dark mode support
4. **Error Handling**: Graceful error display, retry options
5. **Loading States**: Spinners for async operations

## Model Architecture

```go
// Root application model
type App struct {
   // Current view state
   currentView View
   prevViews   []View // Navigation stack
   
   // Shared state
   app         *app.Application
   styles      Styles
   keys        KeyMap
   width       int
   height      int
   
   // Child models (lazy-initialized)
   dashboard   *DashboardModel
   drinks      *DrinksModel
   ingredients *IngredientsModel
   inventory   *InventoryModel
   menus       *MenusModel
   orders      *OrdersModel
   audit       *AuditModel
}

// Each view implements this interface
type ViewModel interface {
   Init() tea.Cmd
   Update(msg tea.Msg) (ViewModel, tea.Cmd)
   View() string
   ShortHelp() []key.Binding
   FullHelp() [][]key.Binding
}

// Message types for cross-view communication
type NavigateMsg struct{ To View }
type RefreshMsg struct{ Entity string }
type ErrorMsg struct{ Err error }
type SelectEntityMsg struct{ Type string; ID string }
```

## Key Bindings

| Key            | Context    | Action              |
|----------------|------------|---------------------|
| `q`, `ctrl+c`  | Global     | Quit                |
| `?`            | Global     | Toggle help         |
| `esc`          | Any view   | Back / Cancel       |
| `1-6`          | Dashboard  | Navigate to section |
| `/`            | List views | Focus search        |
| `f`            | List views | Open filter menu    |
| `c`            | List views | Create new          |
| `enter`        | List views | Edit selected       |
| `d`            | List views | Delete selected     |
| `tab`          | Forms      | Next field          |
| `shift+tab`    | Forms      | Previous field      |
| `ctrl+s`       | Forms      | Save                |
| `↑/↓` or `j/k` | Lists      | Navigate            |
| `g/G`          | Lists      | Go to top/bottom    |

## ID-Free User Experience

The core UX improvement is eliminating ID copy/paste:

| CLI Workflow                                                        | TUI Workflow                                                         |
|---------------------------------------------------------------------|----------------------------------------------------------------------|
| `drinks list` → copy ID → `menu add-drink --menu-id X --drink-id Y` | Navigate to menu → tab to "Available" → select drink → press enter   |
| `ingredients list` → copy ID → `inventory adjust --ingredient-id X` | Navigate to inventory → select ingredient → press `a` → enter amount |
| `menu list` → copy ID → `order place --menu-id X --items Y:2`       | Navigate to orders → select menu from list → add drinks with `+/-`   |

The TUI maintains selection state internally, passing IDs between operations without user intervention.

## Integration with Existing Architecture

The TUI is a new **surface** that consumes the same `app` layer:

```
┌──────────────────────────────────────────────────────────────┐
│                        Surfaces                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐           │
│  │    CLI      │  │    TUI      │  │   (future)  │           │
│  │  urfave/cli │  │  bubbletea  │  │  gRPC/REST  │           │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘           │
│         │                │                │                  │
│         └────────────────┼────────────────┘                  │
│                          ▼                                   │
│  ┌────────────────────────────────────────────────────────┐  │
│  │                    app.Application                     │  │
│  │   Drinks | Ingredients | Inventory | Menu | Orders     │  │
│  └────────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌────────────────────────────────────────────────────────┐  │
│  │                    Infrastructure                      │  │
│  │              Store | Dispatcher | Auth                 │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

## Open Questions

1. **Concurrent Access**: Should TUI auto-refresh data? Poll interval or manual refresh?
2. **Actor Selection**: How to switch between actors (owner/manager/bartender) in TUI?
3. **JSON Import/Export**: Should TUI support paste-from-clipboard for bulk operations?
4. **Error Recovery**: How to handle partial failures in multi-step workflows?
5. **Testing Strategy**: How to test Bubble Tea views? Golden file snapshots?

## Success Criteria

1. All CLI operations available in TUI (feature parity)
2. Zero ID copying required for standard workflows
3. Keyboard-navigable without mouse
4. Responsive to terminal resize (minimum 80x24)
5. Sub-100ms response for local operations
6. Graceful degradation when terminal doesn't support features

## Dependencies

```go
require (
github.com/charmbracelet/bubbletea v1.x
github.com/charmbracelet/bubbles v0.x
github.com/charmbracelet/lipgloss v1.x
)
```

## Estimated Scope

- **New files**: ~15-20 Go files in `main/tui/`
- **Lines of code**: ~2,000-3,000 LOC
- **Sprints**: Implementation can be broken into 5-6 focused sprints following the phases above
