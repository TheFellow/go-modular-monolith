# Sprint 004: Multi-Step Workflows (Saga-Powered)

## Goal

Implement complex multi-step workflows powered by the saga infrastructure from Sprint 003b. Users build up changes in a
saga, preview pending actions, and commit atomically—or discard without side effects. This showcases both the TUI's
usability advantage and the saga pattern's power for managing in-progress work.

## Problem

After Sprint 003, CRUD operations work but still feel like individual operations. The key TUI value proposition—seamless
multi-entity workflows—isn't realized. Additionally, changes persist immediately with no ability to preview, undo, or
abandon work-in-progress.

## Solution

Create specialized workflow views backed by sagas:

1. **Drink Builder** - Build a drink with ingredients in a saga; commit creates drink atomically
2. **Menu Builder** - Curate drinks on a menu; changes accumulate in saga until committed
3. **Order Placement** - Build cart in saga; commit places order atomically
4. **Advanced: Menu Creator** - Define new ingredients, new drinks, and menu in one atomic saga

Each workflow holds a `saga.Saga[T]` that accumulates actions. The user can:
- **Preview**: See what will happen on commit
- **Commit**: Execute all actions atomically
- **Discard**: Abandon without any persistence
- **Undo**: Remove the last action from the saga (while in draft)

## Tasks

### Phase 1: Saga-Aware Workflow Base

- [ ] Create `main/tui/workflow/workflow.go`:
    ```go
    // Workflow wraps a saga with TUI-specific state
    type Workflow[T any] struct {
        Saga      *saga.Saga[T]
        Preview   []string     // Cached action descriptions
        IsDirty   bool         // Has uncommitted changes
        LastError error
    }
    ```
- [ ] Create `main/tui/workflow/messages.go`:
    - `SagaActionAddedMsg` - action added to saga
    - `SagaCommitRequestedMsg` - user wants to commit
    - `SagaCommittedMsg` - commit succeeded
    - `SagaCommitFailedMsg` - commit failed (saga rolled back)
    - `SagaDiscardRequestedMsg` - user wants to discard
    - `SagaDiscardedMsg` - discard completed
    - `SagaUndoMsg` - remove last action
- [ ] Create `main/tui/workflow/preview.go`:
    - `PreviewPanel` component showing pending actions
    - Updates automatically when saga changes
    - Indicates which action is currently executing during commit

### Phase 2: Drink Builder (Saga-Powered)

#### Ingredient Picker Component

- [ ] Create `IngredientPicker` component:
    - Searchable list of all ingredients
    - Filter by category
    - Shows current stock level next to each ingredient
    - Option to define new ingredient inline (adds to saga)
- [ ] Implement `IngredientWithQuantity` struct for local state
- [ ] Create quantity input inline after selecting ingredient

#### Drink Builder View

- [ ] Create `DrinkBuilderModel` backed by `saga.Saga[entity.DrinkID]`:
    ```go
    type DrinkBuilderModel struct {
        workflow    *workflow.Workflow[entity.DrinkID]
        builder     *builders.DrinkBuilder

        // UI components
        nameInput   textinput.Model
        category    SelectField
        glass       SelectField
        price       NumberField
        ingredients IngredientPicker
        preview     PreviewPanel

        // State
        mode        DrinkBuilderMode  // Editing, AddingIngredient, Previewing
    }
    ```
- [ ] Support defining new ingredients during drink creation:
    ```
    ┌─ New Ingredient ──────────────────────────────────────────┐
    │  The ingredient "Blue Curaçao" doesn't exist.             │
    │                                                           │
    │  [Create It]  [Choose Different]  [Cancel]                │
    │                                                           │
    │  If created, it will be added to your saga.               │
    └───────────────────────────────────────────────────────────┘
    ```
- [ ] Show saga preview panel (collapsible):
    ```
    ─── Pending Actions ─────────────────────
    1. Create ingredient: Blue Curaçao
    2. Create drink: Blue Lagoon
    ─────────────────────────────────────────
    [ctrl+s] Commit  [ctrl+d] Discard
    ```

#### Commit/Discard Flow

- [ ] `ctrl+s` or "Save" commits saga:
    - Show confirmation with preview
    - Execute saga
    - On success: Navigate to new drink detail
    - On failure: Show error, saga auto-rolled back, stay in builder
- [ ] `ctrl+d` or "Discard" abandons saga:
    - Confirm if saga has actions: "Discard 2 pending actions?"
    - Clear saga and navigate back
- [ ] `ctrl+z` removes last action from saga (while in draft)

### Phase 3: Menu Builder (Saga-Powered)

#### Two Modes: Live Edit vs. Saga Edit

Menus support two editing modes:

**Live Edit Mode** (existing menu):
- Changes apply immediately (for quick single-drink add/remove)
- Each add/remove is its own mini-saga that auto-commits

**Saga Edit Mode** (new menu or batch edit):
- Changes accumulate in saga
- Preview shows all pending changes
- Explicit commit/discard

- [ ] Add mode toggle: `ctrl+m` switches between Live/Saga mode
- [ ] Visual indicator shows current mode

#### Menu Builder View (Saga Mode)

- [ ] Create `MenuBuilderModel` with saga support:
    ```go
    type MenuBuilderModel struct {
        workflow    *workflow.Workflow[entity.MenuID]
        builder     *builders.MenuBuilder

        // Dual panes
        available   list.Model  // Drinks not on menu (in saga state)
        onMenu      list.Model  // Drinks on menu (in saga state)

        // State derived from saga
        pendingAdds    map[string]bool  // Drinks to be added
        pendingRemoves map[string]bool  // Drinks to be removed

        mode        MenuBuilderMode  // Live or Saga
    }
    ```
- [ ] In saga mode, add/remove updates local state + saga, not server
- [ ] Visual indicators for pending changes:
    - Green `+` badge on drinks pending add
    - Red `-` badge on drinks pending remove
- [ ] Show saga preview panel

#### Define New Drinks While Building Menu

- [ ] "Create Drink" action from within menu builder (`n` key):
    - Opens drink builder as sub-workflow
    - New drink's saga actions merge into menu saga
    - On completion, drink appears in "Available" with green badge
- [ ] Seamless flow: Create ingredient → Create drink → Add to menu (all one saga)

### Phase 4: Order Placement (Saga-Powered)

#### Order Builder View

- [ ] Create `OrderBuilderModel` backed by saga:
    ```go
    type OrderBuilderModel struct {
        workflow    *workflow.Workflow[entity.OrderID]

        menu        *domain.Menu
        cart        OrderCart  // Local state

        // UI
        menuItems   list.Model
        cartList    list.Model
        preview     PreviewPanel
    }
    ```
- [ ] Cart operations update local state only (no saga actions yet)
- [ ] "Place Order" builds saga with single `PlaceOrderAction`:
    ```go
    saga.Add(&orders.PlaceAction{
        MenuID: menu.ID,
        Items:  cart.ToOrderItems(),
        Result: orderRef,
    })
    ```
- [ ] Confirm and commit flow

### Phase 5: Advanced Workflow - Full Menu Creator

Showcase saga power: Create a complete menu from scratch in one atomic operation.

- [ ] Create `FullMenuCreatorModel`:
    ```go
    type FullMenuCreatorModel struct {
        workflow *workflow.Workflow[entity.MenuID]
        builder  *builders.MenuBuilder

        // Wizard steps
        step     int  // 0: Name, 1: Ingredients, 2: Drinks, 3: Review

        // Defined entities (not yet created)
        ingredients map[string]IngredientDraft
        drinks      map[string]DrinkDraft
        menuDrinks  []string  // Keys into drinks map
    }
    ```
- [ ] Step 1: Name menu
- [ ] Step 2: Define any new ingredients needed
- [ ] Step 3: Define drinks (using defined + existing ingredients)
- [ ] Step 4: Select drinks for menu (from defined + existing)
- [ ] Step 5: Review saga preview, showing complete action list:
    ```
    ─── Creating "Summer Specials" Menu ─────
    1. Create ingredient: Aperol
    2. Create ingredient: Prosecco
    3. Create drink: Aperol Spritz (uses: Aperol, Prosecco)
    4. Create menu: Summer Specials
    5. Add drink to menu: Aperol Spritz
    6. Add drink to menu: Mojito (existing)
    7. Add drink to menu: Margarita (existing)
    ─────────────────────────────────────────
    Total: 7 actions

    [Create Menu]  [Edit]  [Discard]
    ```
- [ ] Commit creates everything atomically
- [ ] Failure rolls back all (no orphan ingredients/drinks)

### Phase 6: Workflow State Persistence

- [ ] Persist in-progress sagas to file system:
    - On TUI exit with uncommitted saga: prompt to save draft
    - On TUI launch: check for saved drafts, offer to resume
    - Storage: `~/.local/state/mixology/drafts/`
- [ ] Draft management:
    - List saved drafts on dashboard
    - Resume draft → loads saga and opens appropriate builder
    - Delete draft → discards without executing

### Phase 7: Saga Execution UI

- [ ] Create `CommitProgressModel` for showing saga execution:
    ```
    ─── Committing Saga ─────────────────────

    ✓ Create ingredient: Aperol
    ✓ Create ingredient: Prosecco
    ● Create drink: Aperol Spritz     ← executing
    ○ Create menu: Summer Specials
    ○ Add drink to menu: Aperol Spritz

    ─────────────────────────────────────────
    ```
- [ ] On success: Show summary with created IDs
- [ ] On failure: Show which action failed, what was rolled back

### Phase 8: Quick Actions & Cross-View

- [ ] Dashboard quick actions:
    - `n` - New drink (opens drink builder with fresh saga)
    - `i` - New ingredient (simple saga with one action)
    - `m` - New menu (opens menu builder with fresh saga)
    - `o` - New order (opens order builder)
- [ ] Cross-view actions with saga awareness:
    - From Drinks: `m` adds drink to menu (live mode or prompts for saga)
    - From Menus: `o` starts order from menu
    - From Ingredients: Quick adjust stock (mini-saga)

### Phase 9: Undo Within Workflows

- [ ] Implement undo stack per workflow:
    - Each action added to saga also pushed to undo stack
    - `ctrl+z` pops last action from both saga and undo stack
    - Undo stack cleared on commit or discard
- [ ] Visual feedback: "Undid: Add ingredient Blue Curaçao"

### Phase 10: Workflow Help & Guidance

- [ ] Saga-aware help text:
    - "You have 3 pending actions. Press Ctrl+S to commit or Ctrl+D to discard."
    - "Creating new ingredient will add it to your current saga."
- [ ] First-time tutorial for saga concept:
    - Brief explanation on first workflow entry
    - "Changes aren't saved until you commit. This lets you build complex changes safely."

## Acceptance Criteria

### Drink Builder

- [ ] New drink created via saga (not immediate API call)
- [ ] Can define new ingredients inline (added to same saga)
- [ ] Preview shows all pending actions
- [ ] Commit creates everything atomically
- [ ] Failure rolls back cleanly (no orphan ingredients)
- [ ] Discard abandons without persistence
- [ ] Undo removes last action from saga

### Menu Builder

- [ ] Live mode: Changes apply immediately (existing behavior)
- [ ] Saga mode: Changes accumulate, require explicit commit
- [ ] Visual badges show pending adds/removes
- [ ] Can create new drinks within menu builder (merged into saga)
- [ ] Full atomic commit of complex menu creation

### Order Placement

- [ ] Cart managed locally until commit
- [ ] Single-action saga for order placement
- [ ] Clean confirmation flow

### Full Menu Creator

- [ ] Multi-step wizard creates complete menu
- [ ] Preview shows all actions (ingredients + drinks + menu + assignments)
- [ ] Single commit creates everything
- [ ] Single failure rolls back everything

### Cross-Cutting

- [ ] Sagas persist across TUI restart (draft save/resume)
- [ ] Execution progress shows real-time status
- [ ] All workflows keyboard-navigable
- [ ] Help text explains saga model

## Implementation Details

### Drink Builder with Saga

```go
type DrinkBuilderModel struct {
    workflow *workflow.Workflow[entity.DrinkID]
    builder  *builders.DrinkBuilder

    // Form fields
    name        textinput.Model
    category    string
    glass       string
    price       *float64
    ingredients []IngredientSelection

    // UI state
    previewOpen bool
    styles      DrinkBuilderStyles
}

type IngredientSelection struct {
    // Either existing or to-be-created
    ExistingID *entity.IngredientID
    NewDef     *IngredientDraft
    Quantity   string
}

type IngredientDraft struct {
    Key      string  // Local reference key
    Name     string
    Category string
    Unit     string
}

func (m *DrinkBuilderModel) Init() tea.Cmd {
    // Initialize fresh saga
    m.builder = builders.NewDrinkBuilder(m.name.Value(), m.category, m.glass)
    m.workflow = workflow.New(m.builder.Saga())
    return nil
}

func (m *DrinkBuilderModel) addIngredient(sel IngredientSelection) {
    if sel.ExistingID != nil {
        m.builder.AddExistingIngredient(*sel.ExistingID, sel.Quantity)
    } else {
        // Define new ingredient in saga, then reference it
        m.builder.DefineIngredient(sel.NewDef.Key, commands.CreateIngredient{
            Name:     sel.NewDef.Name,
            Category: sel.NewDef.Category,
            Unit:     sel.NewDef.Unit,
        })
        m.builder.AddIngredient(sel.NewDef.Key, sel.Quantity)
    }
    m.workflow.Refresh()  // Update preview
}

func (m *DrinkBuilderModel) commit() tea.Cmd {
    return func() tea.Msg {
        if err := m.workflow.Saga.Commit(ctx); err != nil {
            return SagaCommitFailedMsg{Err: err}
        }
        return SagaCommittedMsg{Result: m.workflow.Saga.Result}
    }
}
```

### Menu Builder with Dual Mode

```go
type MenuBuilderModel struct {
    menu      *domain.Menu        // nil if creating new
    workflow  *workflow.Workflow[entity.MenuID]
    liveMode  bool                // true = immediate, false = saga

    // Panes
    available list.Model
    onMenu    list.Model

    // Saga state tracking
    pendingAdds    map[string]bool
    pendingRemoves map[string]bool
}

func (m *MenuBuilderModel) addDrink(drink domain.Drink) tea.Cmd {
    if m.liveMode {
        // Immediate - create mini-saga and auto-commit
        s := saga.Of[struct{}]()
        s.Add(&menu.AddDrinkAction{
            MenuID:  m.menu.ID,
            DrinkID: drink.ID,
        })
        return func() tea.Msg {
            if err := s.Commit(ctx); err != nil {
                return ErrorMsg{Err: err}
            }
            return DrinkAddedMsg{DrinkID: drink.ID}
        }
    }

    // Saga mode - accumulate
    m.workflow.Saga.Add(&menu.AddDrinkAction{
        MenuID:  m.menu.ID,
        DrinkID: drink.ID,
    })
    m.pendingAdds[drink.ID.String()] = true
    m.workflow.Refresh()
    return nil
}

func (m *MenuBuilderModel) View() string {
    // Render with pending change indicators
    availableView := m.renderAvailablePane()
    onMenuView := m.renderOnMenuPane()

    panes := lipgloss.JoinHorizontal(lipgloss.Top, availableView, onMenuView)

    if !m.liveMode && len(m.workflow.Preview) > 0 {
        preview := m.renderPreviewPanel()
        return lipgloss.JoinVertical(lipgloss.Left, panes, preview)
    }

    return panes
}

func (m *MenuBuilderModel) renderDrinkItem(d domain.Drink) string {
    name := d.Name
    id := d.ID.String()

    // Add badges for pending changes
    if m.pendingAdds[id] {
        name = m.styles.PendingAdd.Render("+ ") + name
    }
    if m.pendingRemoves[id] {
        name = m.styles.PendingRemove.Render("- ") + name
    }

    return name
}
```

### Commit Progress View

```go
type CommitProgressModel struct {
    actions     []string
    currentIdx  int
    completed   []bool
    failed      int  // -1 if none failed
    err         error
    done        bool
}

func (m *CommitProgressModel) View() string {
    var b strings.Builder
    b.WriteString("─── Committing Saga ─────────────────────\n\n")

    for i, action := range m.actions {
        var prefix string
        switch {
        case i < m.currentIdx:
            prefix = m.styles.Completed.Render("✓")
        case i == m.currentIdx:
            prefix = m.styles.Current.Render("●")
        case i == m.failed:
            prefix = m.styles.Failed.Render("✗")
        default:
            prefix = m.styles.Pending.Render("○")
        }

        line := fmt.Sprintf("%s %s", prefix, action)
        if i == m.currentIdx {
            line = m.styles.Highlight.Render(line)
        }
        b.WriteString(line + "\n")
    }

    b.WriteString("\n─────────────────────────────────────────\n")

    if m.err != nil {
        b.WriteString(m.styles.Error.Render(fmt.Sprintf("Failed: %v", m.err)))
    }

    return b.String()
}
```

## Notes

### When to Use Sagas vs. Immediate

| Operation | Mode | Rationale |
|-----------|------|-----------|
| Quick single add/remove | Immediate (auto-commit mini-saga) | Fast feedback for simple ops |
| New drink with ingredients | Full saga | May need to define ingredients |
| New menu from scratch | Full saga | Complex multi-entity operation |
| Inventory adjustment | Immediate (single action) | Simple, reversible |
| Order placement | Full saga | Should preview before committing |

### Saga as Source of Truth

During saga-mode workflows, the saga (not the server) is the source of truth for the UI. The "On Menu" pane shows the
server state + pending adds - pending removes.

### Error Recovery

If saga commit fails:
1. Saga automatically rolls back
2. User stays in builder with all their work intact
3. Error message explains what failed
4. User can edit and retry

### Performance

Sagas accumulate actions in memory. For very large sagas (100+ actions), consider pagination or warnings. In practice,
typical workflows have <20 actions.
