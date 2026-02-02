# Task 002: Confirmation Dialog Component

## Goal

Create a reusable confirmation dialog in `pkg/tui/dialog/` for delete confirmations and other dangerous actions.

## Files to Create

```
pkg/tui/dialog/
├── confirm.go    # ConfirmDialog model
├── styles.go     # DialogStyles type
└── keys.go       # DialogKeys type
```

## Implementation

### ConfirmDialog

```go
// pkg/tui/dialog/confirm.go
package dialog

type ConfirmDialog struct {
    title       string
    message     string
    confirmBtn  string
    cancelBtn   string
    dangerous   bool  // Red confirm button style
    focused     int   // 0=confirm, 1=cancel
    confirmed   bool
    cancelled   bool
    styles      DialogStyles
    keys        DialogKeys
    width       int
}

// Messages
type ConfirmMsg struct{}
type CancelMsg struct{}

func NewConfirmDialog(title, message string, opts ...DialogOption) *ConfirmDialog

// DialogOption pattern
type DialogOption func(*ConfirmDialog)

func WithConfirmText(text string) DialogOption  // Default: "Confirm"
func WithCancelText(text string) DialogOption   // Default: "Cancel"
func WithDangerous() DialogOption               // Red confirm button
func WithFocusCancel() DialogOption             // Focus cancel by default (safer)

func (d *ConfirmDialog) Init() tea.Cmd
func (d *ConfirmDialog) Update(msg tea.Msg) (*ConfirmDialog, tea.Cmd)
func (d *ConfirmDialog) View() string
func (d *ConfirmDialog) SetWidth(w int)

// State queries
func (d *ConfirmDialog) IsConfirmed() bool
func (d *ConfirmDialog) IsCancelled() bool
```

### Usage Pattern

```go
// In a ViewModel handling delete
case tea.KeyMsg:
    if key.Matches(msg, keys.Delete) {
        return m, showConfirmDialog(
            "Delete Ingredient",
            "Delete 'Vodka'? This will also delete 3 drinks.",
            dialog.WithDangerous(),
            dialog.WithFocusCancel(),
        )
    }

// Handle dialog result
case dialog.ConfirmMsg:
    return m, m.performDelete()
case dialog.CancelMsg:
    m.dialog = nil
    return m, nil
```

### Styles and Keys

```go
// pkg/tui/dialog/styles.go
package dialog

type DialogStyles struct {
    Modal         lipgloss.Style  // Outer box
    Title         lipgloss.Style
    Message       lipgloss.Style
    Button        lipgloss.Style
    ButtonFocused lipgloss.Style
    DangerButton  lipgloss.Style
}

// pkg/tui/dialog/keys.go
package dialog

type DialogKeys struct {
    Confirm  key.Binding  // enter
    Cancel   key.Binding  // esc
    Switch   key.Binding  // tab / left / right
}
```

### Rendering

The dialog renders as a centered modal:

```
┌─────────────────────────────────────┐
│           Delete Ingredient         │
│                                     │
│  Delete "Vodka"?                    │
│                                     │
│  This will also delete 3 drinks    │
│  that use this ingredient.          │
│                                     │
│         [Delete]    [Cancel]        │
└─────────────────────────────────────┘
```

- Modal is centered in available space
- Title at top
- Message body with word wrap
- Buttons at bottom
- Focused button is highlighted/underlined
- Dangerous confirm button has error/red styling

## Notes

- Keep dialog simple - just confirm/cancel
- Parent ViewModel is responsible for overlay rendering
- Dialog sends messages back, doesn't call app directly
- Focus cancel by default for dangerous operations (WithFocusCancel)

## Checklist

- [x] Create `pkg/tui/dialog/` directory
- [x] Implement `confirm.go` with ConfirmDialog
- [x] Implement `styles.go` with DialogStyles
- [x] Implement `keys.go` with DialogKeys
- [x] Add tests for keyboard navigation and message sending
- [x] `go build ./pkg/tui/dialog/...` passes
- [x] `go test ./pkg/tui/dialog/...` passes
