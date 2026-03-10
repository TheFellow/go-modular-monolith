# Sprint 004: Multi-Step Workflows (Saga-Powered)

## Goal

Implement complex multi-step workflows powered by the saga infrastructure from Sprint 003b. Users build up changes in a
saga, preview pending operations, and commit atomically—or cancel without side effects. This showcases both the TUI's
usability advantage and the saga pattern's power for managing in-progress work.

## Problem

After Sprint 003, CRUD operations work but still feel like individual operations. The key TUI value proposition—seamless
multi-entity workflows—isn't realized. Additionally, changes persist immediately with no ability to preview or
abandon work-in-progress.

## Solution

Create specialized workflow views backed by sagas:

1. **Drink Creation** - Build a drink with ingredients in a saga; commit creates drink atomically
2. **Menu Creation** - Curate drinks on a menu; changes accumulate in saga until committed
3. **Order Placement** - Build cart in saga; commit places order atomically
4. **Advanced: Menu Creator** - Define new ingredients, new drinks, and menu in one atomic saga

Each workflow holds a `saga.Saga[T]` that accumulates data. The user can:
- **Preview**: See what will happen on commit
- **Commit**: Execute all operations atomically
- **Cancel**: Navigate away (saga is garbage collected, no persistence)
- **Edit/Delete**: Modify or remove items from the saga before committing

## Tasks

### Phase 1: Composable ViewModel Architecture

The key insight: **ViewModels should be decoupled from concrete saga types**. This allows:
- `ingredients.CreateViewModel` to work standalone OR nested in `drinks.CreateViewModel`
- `drinks.CreateViewModel` to work standalone OR nested in `menus.CreateViewModel`
- The same UI logic to contribute to different sagas

#### Capability Interfaces

- [ ] Create `pkg/saga/capabilities.go`:
    ```go
    // Capability interfaces - sagas implement these to accept ViewModel contributions
    // These live in pkg/saga because they define the contract between sagas and TUI ViewModels

    // IngredientDefiner is implemented by sagas that can create ingredients
    type IngredientDefiner interface {
        DefineIngredient(key string, def IngredientDef)
        Preview() []string
    }

    // DrinkDefiner is implemented by sagas that can create drinks
    type DrinkDefiner interface {
        IngredientDefiner  // Drinks may need new ingredients
        DefineDrink(key string, def DrinkDef)
        AddExistingIngredient(drinkKey string, ingredientID entity.IngredientID, qty string)
        AddIngredientByKey(drinkKey string, ingredientKey string, qty string)
    }

    // MenuDrinkAdder is implemented by sagas that can add drinks to menus
    type MenuDrinkAdder interface {
        DrinkDefiner  // Menus may need new drinks (which may need new ingredients)
        AddDrinkByKey(drinkKey string, price *float64)
        AddExistingDrink(drinkID entity.DrinkID, price *float64)
    }
    ```

- [ ] Verify saga types implement capabilities:
    ```go
    // These compile-time checks ensure sagas implement the right interfaces
    var _ saga.IngredientDefiner = (*IngredientSaga)(nil)  // in ingredients/saga.go
    var _ saga.DrinkDefiner = (*DrinkSaga)(nil)            // in drinks/saga.go
    var _ saga.MenuDrinkAdder = (*MenuSaga)(nil)           // in menus/saga.go
    ```

#### Domain TUI Surfaces

Each domain owns its TUI ViewModels under `surfaces/tui/`:

- [ ] Create `app/domains/ingredients/surfaces/tui/create_vm.go`:
    ```go
    // CreateViewModel is a reusable ViewModel for defining an ingredient
    // It works with ANY saga that implements saga.IngredientDefiner
    type CreateViewModel struct {
        definer saga.IngredientDefiner  // The backing saga (any type)

        // Form fields
        name     textinput.Model
        category SelectField
        unit     SelectField

        // State
        key    string  // Local key for this ingredient definition
        styles Styles
    }

    // NewCreateViewModel creates a ViewModel that contributes to any IngredientDefiner
    func NewCreateViewModel(definer saga.IngredientDefiner, key string) *CreateViewModel {
        return &CreateViewModel{
            definer: definer,
            key:     key,
            // ... init form fields
        }
    }

    // Submit adds the ingredient definition to the backing saga
    func (vm *CreateViewModel) Submit() tea.Cmd {
        vm.definer.DefineIngredient(vm.key, saga.IngredientDef{
            Name:     vm.name.Value(),
            Category: vm.category.Value(),
            Unit:     vm.unit.Value(),
        })
        return func() tea.Msg { return IngredientDefinedMsg{Key: vm.key} }
    }
    ```

- [ ] Create `app/domains/drinks/surfaces/tui/create_vm.go`:
    ```go
    // CreateViewModel is a reusable ViewModel for defining a drink
    // It works with ANY saga that implements saga.DrinkDefiner
    type CreateViewModel struct {
        definer saga.DrinkDefiner  // The backing saga (any type)

        // Form fields
        name     textinput.Model
        category SelectField
        glass    SelectField
        price    NumberField

        // Nested ViewModel for inline ingredient creation
        ingredientVM *ingredients.CreateViewModel

        // Ingredients added to this drink
        ingredients []IngredientSelection

        // State
        key  string
        mode CreateMode  // Editing, AddingIngredient
    }

    // When user needs a new ingredient, we create a nested ViewModel
    // that contributes to the SAME backing saga
    func (vm *CreateViewModel) startAddIngredient() {
        key := fmt.Sprintf("ing_%d", len(vm.ingredients))
        vm.ingredientVM = ingredients.NewCreateViewModel(vm.definer, key)
        vm.mode = AddingIngredient
    }
    ```

#### Workflow Messages

- [ ] Create `main/tui/workflow/messages.go`:
    - `SagaCommitRequestedMsg` - user wants to commit
    - `SagaCommittedMsg[T]` - commit succeeded, contains result
    - `SagaCommitFailedMsg` - commit failed (transaction rolled back)
    - `IngredientDefinedMsg` - ingredient added to saga
    - `DrinkDefinedMsg` - drink added to saga
    - `ViewModelPushedMsg` - nested ViewModel activated
    - `ViewModelPoppedMsg` - nested ViewModel completed/cancelled

#### Preview & Commit Helpers

- [ ] Create `main/tui/workflow/preview.go`:
    - `PreviewPanel` component showing pending actions
    - Takes `[]string` from any saga's `Preview()` method
    - Indicates which action is currently executing during commit

- [ ] Create `main/tui/workflow/commit.go`:
    ```go
    // CommitSaga executes a saga through the middleware chain
    func CommitSaga[T any](ctx *middleware.Context, s saga.Saga[T]) tea.Cmd {
        return func() tea.Msg {
            result, err := middleware.Saga.Execute(ctx, s)
            if err != nil {
                return SagaCommitFailedMsg{Err: err}
            }
            return SagaCommittedMsg[T]{Result: result}
        }
    }
    ```

### Phase 2: Ingredient CreateViewModel (Standalone & Composable)

The `CreateViewModel` works in two contexts:
1. **Standalone**: Backed by `IngredientSaga`, commits to create one ingredient
2. **Nested**: Backed by a parent saga (DrinkSaga, MenuSaga), contributes to parent's transaction

#### Standalone Usage

When used standalone, the view owns the saga:

- [ ] In `app/domains/ingredients/surfaces/tui/create_vm.go`, add standalone factory:
    ```go
    // NewStandaloneCreateViewModel creates a ViewModel with its own IngredientSaga
    // Use this for standalone ingredient creation (not nested in another workflow)
    func NewStandaloneCreateViewModel(ctx *middleware.Context, app *app.Application) *CreateViewModel {
        saga := NewIngredientSaga(app)
        return &CreateViewModel{
            definer: saga,
            saga:    saga,  // We own this saga
            ctx:     ctx,
            key:     "_primary",
        }
    }

    func (vm *CreateViewModel) Commit() tea.Cmd {
        vm.Submit()
        return workflow.CommitSaga(vm.ctx, vm.saga)
    }
    ```

- [ ] Wire `c` key in Ingredients list to open `CreateViewModel`
- [ ] On commit success, navigate to new ingredient in list

### Phase 3: Drink CreateViewModel (Saga-Powered)

The `CreateViewModel` works in two contexts:
1. **Standalone**: Backed by `DrinkSaga`, commits to create one drink (with optional new ingredients)
2. **Nested**: Backed by a parent `MenuSaga`, contributes drinks to the menu's transaction

#### Ingredient Picker Component

- [ ] Create `IngredientPicker` component:
    - Searchable list of all ingredients
    - Filter by category
    - Shows current stock level next to each ingredient
    - Option to define new ingredient inline (adds to saga)
- [ ] Implement `IngredientWithQuantity` struct for local state
- [ ] Create quantity input inline after selecting ingredient

#### Drink CreateViewModel (Reusable Component)

Already defined in Phase 1 under `app/domains/drinks/surfaces/tui/create_vm.go`.

Key methods:
```go
// When user needs a new ingredient, spawn nested ViewModel
// The nested ViewModel contributes to the SAME backing saga
func (vm *CreateViewModel) startAddIngredient() tea.Cmd {
    key := fmt.Sprintf("%s_ing_%d", vm.key, len(vm.ingredients))
    vm.ingredientVM = ingredients.NewCreateViewModel(vm.definer, key)
    vm.mode = AddingIngredient
    return func() tea.Msg { return ViewModelPushedMsg{Type: "ingredient"} }
}

// After ingredient ViewModel completes, add it to this drink
func (vm *CreateViewModel) handleIngredientDefined(msg IngredientDefinedMsg) {
    vm.ingredients = append(vm.ingredients, IngredientSelection{
        Key:      msg.Key,
        Quantity: vm.ingredientVM.LastQuantity(),
    })
    vm.ingredientVM = nil
    vm.mode = Editing
}
```

#### Standalone Usage

- [ ] In `app/domains/drinks/surfaces/tui/create_vm.go`, add standalone factory:
    ```go
    // NewStandaloneCreateViewModel creates a ViewModel with its own DrinkSaga
    func NewStandaloneCreateViewModel(ctx *middleware.Context, app *app.Application) *CreateViewModel {
        saga := NewDrinkSaga(app)
        return &CreateViewModel{
            definer: saga,
            saga:    saga,  // We own this saga
            ctx:     ctx,
            key:     "_primary",
        }
    }

    func (vm *CreateViewModel) Commit() tea.Cmd {
        vm.Submit()
        return workflow.CommitSaga(vm.ctx, vm.saga)
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
    [ctrl+s] Commit  [esc] Cancel
    ```

#### Commit/Cancel Flow

- [ ] `ctrl+s` or "Save" commits saga:
    - Show confirmation with preview
    - Execute saga via `workflow.CommitSaga(ctx, saga)`
    - On success: Navigate to new drink detail
    - On failure: Show error, transaction rolled back, stay in CreateViewModel
- [ ] `esc` cancels and navigates back:
    - Confirm if saga has pending operations: "Cancel? 2 pending operations will be lost."
    - Navigate back (saga is garbage collected)

### Phase 4: Menu CreateViewModel (Saga-Powered)

#### Two Modes: Live Edit vs. Saga Edit

Menus support two editing modes:

**Live Edit Mode** (existing menu):
- Changes apply immediately (for quick single-drink add/remove)
- Each add/remove uses the command chain directly

**Saga Edit Mode** (new menu or batch edit):
- Changes accumulate in saga
- Preview shows all pending changes
- Explicit commit/cancel

- [ ] Add mode toggle: `ctrl+m` switches between Live/Saga mode
- [ ] Visual indicator shows current mode

#### Menu CreateViewModel (Saga Mode)

- [ ] Create `app/domains/menus/surfaces/tui/create_vm.go`:
    ```go
    // CreateViewModel owns the saga and orchestrates nested ViewModels from other domains
    type CreateViewModel struct {
        ctx   *middleware.Context
        app   *app.Application
        menu  *domain.Menu   // nil if creating new
        saga  *MenuSaga      // MenuSaga implements saga.MenuDrinkAdder

        // Nested ViewModels from other domains (contribute to same saga)
        drinkVM      *drinks.CreateViewModel
        ingredientVM *ingredients.CreateViewModel

        // UI components
        nameInput   textinput.Model  // For new menus
        available   list.Model       // Drinks not on menu
        onMenu      list.Model       // Drinks on menu

        // State derived from saga (for UI rendering)
        pendingAdds    map[string]bool
        pendingRemoves map[string]bool

        mode  CreateMode  // Live or Saga
        focus FocusState  // Main, DrinkVM, IngredientVM
    }

    // "Create Drink" opens nested drink ViewModel backed by SAME saga
    func (vm *CreateViewModel) startCreateDrink() tea.Cmd {
        key := fmt.Sprintf("drink_%d", vm.drinkCount)
        vm.drinkVM = drinks.NewCreateViewModel(vm.saga, key)
        vm.focus = DrinkVMFocus
        return func() tea.Msg { return ViewModelPushedMsg{Type: "drink"} }
    }

    // When drink ViewModel completes, add it to menu
    func (vm *CreateViewModel) handleDrinkDefined(msg DrinkDefinedMsg) {
        vm.saga.AddDrinkByKey(msg.Key, nil)  // Add to menu at default price
        vm.pendingAdds[msg.Key] = true
        vm.drinkVM = nil
        vm.focus = MainFocus
    }
    ```

- [ ] In saga mode, add/remove updates saga, not server
- [ ] Visual indicators for pending changes:
    - Green `+` badge on drinks pending add
    - Red `-` badge on drinks pending remove
- [ ] Show saga preview panel

#### Nested ViewModel Flow

- [ ] "Create Drink" action (`n` key):
    - Spawns `drinks.CreateViewModel` backed by the **same MenuSaga**
    - Drink ViewModel can spawn `ingredients.CreateViewModel` (also same saga)
    - All definitions accumulate in one saga
    - On completion, drink appears in "Available" with green badge

- [ ] ViewModel stack navigation:
    - `esc` pops current ViewModel (with confirmation if pending operations)
    - Completion auto-pops and notifies parent
    - Preview always shows full saga state across all nested ViewModels

- [ ] Seamless flow example:
    ```
    menus.CreateVM → [n] → drinks.CreateVM → [i] → ingredients.CreateVM
                                                     ↓ submit
                                              drinks.CreateVM (ingredient added)
                                                     ↓ submit
                                              menus.CreateVM (drink added)
                                                     ↓ commit
                                              All created atomically!
    ```

### Phase 5: Order Placement (Saga-Powered)

#### Order CreateViewModel

- [ ] Create `OrderCreateViewModel` backed by saga:
    ```go
    type OrderCreateViewModel struct {
        ctx         *middleware.Context
        app         *app.Application
        saga        *orders.OrderSaga

        menu        *domain.Menu
        cart        OrderCart  // Local state for UI

        // UI
        menuItems   list.Model
        cartList    list.Model
    }
    ```
- [ ] Cart operations update local state (for UI) and saga simultaneously:
    ```go
    func (vm *OrderCreateViewModel) addToCart(drink domain.Drink, qty int) {
        vm.cart.Add(drink, qty)
        vm.saga.AddItem(orders.OrderItem{DrinkID: drink.ID, Quantity: qty})
    }
    ```
- [ ] "Place Order" commits through middleware:
    ```go
    func (vm *OrderCreateViewModel) placeOrder() tea.Cmd {
        return workflow.CommitSaga(vm.ctx, vm.saga)
    }
    ```
- [ ] Confirm and commit flow

### Phase 6: Advanced Workflow - Full Menu Creator

Showcase saga power: Create a complete menu from scratch in one atomic operation.

- [ ] Create `FullMenuCreatorModel`:
    ```go
    type FullMenuCreatorModel struct {
        ctx   *middleware.Context
        app   *app.Application
        saga  *menus.MenuSaga

        // Wizard steps
        step  int  // 0: Name, 1: Ingredients, 2: Drinks, 3: Review
    }

    func NewFullMenuCreator(ctx *middleware.Context, app *app.Application) *FullMenuCreatorModel {
        return &FullMenuCreatorModel{
            ctx:  ctx,
            app:  app,
            saga: menus.NewMenuSaga(app),
        }
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

    [Create Menu]  [Edit]  [Cancel]
    ```
- [ ] Commit creates everything atomically
- [ ] Failure rolls back all (no orphan ingredients/drinks)

### Phase 7: Workflow State Persistence

- [ ] Persist in-progress sagas to file system:
    - On TUI exit with uncommitted saga: prompt to save draft
    - On TUI launch: check for saved drafts, offer to resume
    - Storage: `~/.local/state/mixology/drafts/`
- [ ] Draft management:
    - List saved drafts on dashboard
    - Resume draft → loads saga and opens appropriate CreateViewModel
    - Delete draft → removes without executing

### Phase 8: Saga Execution UI

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

### Phase 9: Quick Actions & Cross-View

- [ ] Dashboard quick actions:
    - `n` - New drink (opens drink CreateViewModel with fresh saga)
    - `i` - New ingredient (simple saga with one action)
    - `m` - New menu (opens menu CreateViewModel with fresh saga)
    - `o` - New order (opens order CreateViewModel)
- [ ] Cross-view actions with saga awareness:
    - From Drinks: `m` adds drink to menu (live mode or prompts for saga)
    - From Menus: `o` starts order from menu
    - From Ingredients: Quick adjust stock (mini-saga)

### Phase 10: Workflow Help & Guidance

- [ ] Saga-aware help text:
    - "You have 3 pending operations. Press Ctrl+S to commit or Esc to cancel."
    - "Creating new ingredient will add it to your current saga."
- [ ] First-time tutorial for saga concept:
    - Brief explanation on first workflow entry
    - "Changes aren't saved until you commit. This lets you build complex changes safely."

## Acceptance Criteria

### Drink CreateViewModel

- [ ] New drink created via saga (not immediate API call)
- [ ] Can define new ingredients inline (added to same saga)
- [ ] Preview shows all pending actions
- [ ] Commit creates everything atomically
- [ ] Failure rolls back cleanly (no orphan ingredients)
- [ ] Cancel abandons without persistence (saga is garbage collected)

### Menu CreateViewModel

- [ ] Live mode: Changes apply immediately (existing behavior)
- [ ] Saga mode: Changes accumulate, require explicit commit
- [ ] Visual badges show pending adds/removes
- [ ] Can create new drinks within menu CreateViewModel (merged into saga)
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

### Composable CreateViewModel Pattern

```go
// drinks/surfaces/tui/create_vm.go

// CreateViewModel is a REUSABLE ViewModel
// It works with ANY saga implementing DrinkDefiner
type CreateViewModel struct {
    definer saga.DrinkDefiner  // Could be DrinkSaga or MenuSaga
    key     string             // Local key for this drink

    // Form fields
    name        textinput.Model
    category    string
    glass       string
    price       *float64

    // Nested ingredient ViewModel (same backing saga)
    ingredientVM *ingredients.CreateViewModel

    // Ingredients for this drink
    ingredients []IngredientSelection

    // For standalone use only (nil when nested)
    saga *DrinkSaga
    ctx  *middleware.Context

    mode CreateMode
}

// NewCreateViewModel creates a ViewModel that contributes to any DrinkDefiner
func NewCreateViewModel(definer saga.DrinkDefiner, key string) *CreateViewModel {
    return &CreateViewModel{
        definer: definer,
        key:     key,
        // ... init form fields
    }
}

// NewStandaloneCreateViewModel creates a ViewModel with its own saga
func NewStandaloneCreateViewModel(ctx *middleware.Context, app *app.Application) *CreateViewModel {
    saga := NewDrinkSaga(app)
    return &CreateViewModel{
        definer: saga,
        saga:    saga,
        ctx:     ctx,
        key:     "_primary",
    }
}

// Start nested ingredient creation (contributes to SAME saga)
func (vm *CreateViewModel) startAddIngredient() {
    ingKey := fmt.Sprintf("%s_ing_%d", vm.key, len(vm.ingredients))
    vm.ingredientVM = ingredients.NewCreateViewModel(vm.definer, ingKey)
    vm.mode = AddingIngredient
}

// Submit defines the drink in the backing saga
func (vm *CreateViewModel) Submit() tea.Cmd {
    vm.definer.DefineDrink(vm.key, saga.DrinkDef{
        Name:     vm.name.Value(),
        Category: vm.category,
        Glass:    vm.glass,
        Price:    vm.price,
    })
    // Add ingredient references
    for _, ing := range vm.ingredients {
        if ing.Key != "" {
            vm.definer.AddIngredientByKey(vm.key, ing.Key, ing.Quantity)
        } else {
            vm.definer.AddExistingIngredient(vm.key, ing.ExistingID, ing.Quantity)
        }
    }
    return func() tea.Msg { return DrinkDefinedMsg{Key: vm.key} }
}

// Commit is only valid for standalone ViewModels
func (vm *CreateViewModel) Commit() tea.Cmd {
    vm.Submit()
    return workflow.CommitSaga(vm.ctx, vm.saga)
}
```

### Menu CreateViewModel with Nested ViewModels

```go
// menus/surfaces/tui/create_vm.go

// CreateViewModel owns the saga and orchestrates nested ViewModels
type CreateViewModel struct {
    ctx   *middleware.Context
    app   *app.Application
    menu  *domain.Menu      // nil if creating new
    saga  *MenuSaga         // Implements MenuDrinkAdder (which extends DrinkDefiner)

    // Nested ViewModels (contribute to SAME saga)
    drinkVM      *drinks.CreateViewModel
    ingredientVM *ingredients.CreateViewModel

    // UI components
    nameInput  textinput.Model
    available  list.Model
    onMenu     list.Model

    // State derived from saga
    pendingAdds    map[string]bool
    pendingRemoves map[string]bool

    focus FocusState  // Main, DrinkVM, IngredientVM
}

// Start nested drink ViewModel (contributes to same saga)
func (vm *CreateViewModel) startCreateDrink() tea.Cmd {
    key := fmt.Sprintf("drink_%d", vm.drinkCount)
    // drinks.CreateViewModel works with MenuSaga because MenuSaga implements DrinkDefiner
    vm.drinkVM = drinks.NewCreateViewModel(vm.saga, key)
    vm.focus = DrinkVMFocus
    return func() tea.Msg { return ViewModelPushedMsg{Type: "drink"} }
}

// Handle completion of nested drink ViewModel
func (vm *CreateViewModel) handleDrinkDefined(msg DrinkDefinedMsg) {
    // Drink is now defined in saga, add it to menu
    vm.saga.AddDrinkByKey(msg.Key, nil)
    vm.pendingAdds[msg.Key] = true
    vm.drinkVM = nil
    vm.focus = MainFocus
}

// Add existing drink directly
func (vm *CreateViewModel) addExistingDrink(drink domain.Drink, price *float64) {
    vm.saga.AddExistingDrink(drink.ID, price)
    vm.pendingAdds[drink.ID.String()] = true
}

func (vm *CreateViewModel) Commit() tea.Cmd {
    return workflow.CommitSaga(vm.ctx, vm.saga)
}

func (vm *CreateViewModel) View() string {
    // If nested ViewModel is active, delegate to it
    switch vm.focus {
    case DrinkVMFocus:
        return vm.drinkVM.View()
    case IngredientVMFocus:
        return vm.ingredientVM.View()
    }

    // Main view with pending change indicators
    availableView := vm.renderAvailablePane()
    onMenuView := vm.renderOnMenuPane()
    panes := lipgloss.JoinHorizontal(lipgloss.Top, availableView, onMenuView)

    // Show preview panel if there are pending operations
    if steps := vm.saga.Preview(); len(steps) > 0 {
        preview := vm.renderPreviewPanel(steps)
        return lipgloss.JoinVertical(lipgloss.Left, panes, preview)
    }

    return panes
}

func (vm *CreateViewModel) renderPreviewPanel(steps []string) string {
    // Preview shows ALL pending operations across nested ViewModels
    // ... render steps
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

### MVVM-Aligned Architecture

The pattern follows MVVM (Model-View-ViewModel), adapted for Bubble Tea:

| MVVM          | Our Pattern     | Responsibility                                                       |
|---------------|-----------------|----------------------------------------------------------------------|
| **Model**     | Saga            | Domain data, validation, execution logic                             |
| **ViewModel** | CreateViewModel | UI state, adapts Model for View, works against capability interfaces |
| **View**      | `View()` output | Bubble Tea rendering                                                 |

```
┌─────────────────────────────────────────────────────────────────────────┐
│  Model Layer (pkg/saga/)                                                │
├─────────────────────────────────────────────────────────────────────────┤
│  Capability Interfaces:                                                 │
│  IngredientDefiner ──► DrinkDefiner ──► MenuDrinkAdder                  │
│        ▲                    ▲                 ▲                         │
│        │                    │                 │                         │
│  IngredientSaga        DrinkSaga         MenuSaga                       │
├─────────────────────────────────────────────────────────────────────────┤
│  ViewModel Layer (surfaces/tui/)                                        │
├─────────────────────────────────────────────────────────────────────────┤
│  ingredients/         drinks/            menus/                         │
│  CreateViewModel      CreateViewModel    CreateViewModel                │
│  ListViewModel        ListViewModel      ListViewModel                  │
│  DetailViewModel      DetailViewModel    DetailViewModel                │
└─────────────────────────────────────────────────────────────────────────┘
```

**Why this enables reuse:**
- `CreateViewModel`s work against **capability interfaces**, not concrete types
- `ingredients.CreateViewModel` works with any `IngredientDefiner` (IngredientSaga, DrinkSaga, or MenuSaga)
- `drinks.CreateViewModel` works with any `DrinkDefiner` (DrinkSaga or MenuSaga)
- All nested ViewModels contribute to the **same Model (saga)** for atomic commits
- Parent ViewModel owns the saga; child ViewModels just hold a reference to the interface

### Directory Structure

Each domain owns its TUI surface components with consistent ViewModel naming:

```
pkg/saga/
├── saga.go           # Saga[T] interface
├── permission.go     # Permission type
├── resolution.go     # Resolution for tracking created IDs
├── authorize.go      # AuthorizeAll, DeduplicatePermissions
└── capabilities.go   # IngredientDefiner, DrinkDefiner, MenuDrinkAdder

app/domains/ingredients/
├── saga.go                    # IngredientSaga (Model)
└── surfaces/tui/
    ├── list_vm.go             # ListViewModel - read-only list display
    ├── detail_vm.go           # DetailViewModel - read-only detail display
    └── create_vm.go           # CreateViewModel - saga-backed, reusable

app/domains/drinks/
├── saga.go                    # DrinkSaga (Model)
└── surfaces/tui/
    ├── list_vm.go             # ListViewModel
    ├── detail_vm.go           # DetailViewModel
    └── create_vm.go           # CreateViewModel - can nest ingredients.CreateViewModel

app/domains/menus/
├── saga.go                    # MenuSaga (Model)
└── surfaces/tui/
    ├── list_vm.go             # ListViewModel
    ├── detail_vm.go           # DetailViewModel
    └── create_vm.go           # CreateViewModel - orchestrates nested ViewModels

main/tui/
├── main.go                    # Entry point
├── app.go                     # Root model, navigation
├── styles.go                  # Lip Gloss styles
├── keys.go                    # Key bindings
├── messages.go                # Shared navigation messages
└── workflow/
    ├── messages.go            # SagaCommittedMsg, etc.
    ├── preview.go             # PreviewPanel component
    └── commit.go              # CommitSaga helper
```

**ViewModel types:**

| ViewModel | Saga-backed? | Purpose |
|-----------|--------------|---------|
| `ListViewModel` | No | Query and display list, handle filtering/selection |
| `DetailViewModel` | No | Query and display single entity |
| `CreateViewModel` | Yes | Saga-backed creation workflow, reusable via capability interfaces |
| `EditViewModel` | Yes | Saga-backed editing (often reuses CreateViewModel with existing data) |

### When to Use Sagas vs. Immediate

| Operation                  | Mode                      | Rationale                        |
|----------------------------|---------------------------|----------------------------------|
| Quick single add/remove    | Immediate (command chain) | Fast feedback for simple ops     |
| New drink with ingredients | Full saga                 | May need to define ingredients   |
| New menu from scratch      | Full saga                 | Complex multi-entity operation   |
| Inventory adjustment       | Immediate (command chain) | Simple, reversible               |
| Order placement            | Full saga                 | Should preview before committing |

### Saga as Source of Truth

During saga-mode workflows, the saga (not the server) is the source of truth for the UI. The "On Menu" pane shows the
server state + pending adds - pending removes. Use `saga.Preview()` to show what will happen on commit.

### Middleware Execution

All saga commits go through `middleware.Saga.Execute(ctx, saga)`, which:
1. Validates the saga data
2. Pre-checks all required permissions (fails fast before transaction)
3. Wraps execution in a database transaction
4. Commits on success, rolls back on any failure

### Error Recovery

If saga commit fails:
1. Transaction automatically rolls back (no partial state)
2. User stays in CreateViewModel with all their work intact
3. Error message explains what failed (validation, permission, or execution error)
4. User can edit and retry

### Performance

Sagas accumulate data in memory. For very large sagas (100+ operations), consider pagination or warnings. In practice,
typical workflows have <20 operations.
