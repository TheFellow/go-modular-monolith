# Task 002: Create Shared Message Types

## Goal

Define the message types used for communication between views and the root App model.

## File to Create

`main/tui/messages.go`

## Pattern Reference

Bubble Tea uses message passing for state changes. Messages are plain Go types that flow through the `Update` function.

## Implementation

Create a new file with the following message types:

```go
package tui

// NavigateMsg requests navigation to a different view
type NavigateMsg struct {
    To View
}

// BackMsg requests navigation to the previous view
type BackMsg struct{}

// ErrorMsg carries an error to display in the status bar
type ErrorMsg struct {
    Err error
}

// RefreshMsg requests the current view to reload its data
type RefreshMsg struct{}

// View represents a navigable view in the TUI
type View int

const (
    ViewDashboard View = iota
    ViewDrinks
    ViewIngredients
    ViewInventory
    ViewMenus
    ViewOrders
    ViewAudit
)

// String returns the display name for the view
func (v View) String() string {
    // Implement switch statement
}

// ParseView converts a string argument to a View
func ParseView(s string) (View, bool) {
    // Implement lookup
}
```

## Notes

- The `View` type is defined here since messages reference it
- `ParseView` will be used for `--tui <view>` argument parsing
- This file has no dependencies on other TUI files, making it safe to create first

## Checklist

- [x] Create `main/tui/messages.go`
- [x] Define all message types (NavigateMsg, BackMsg, ErrorMsg, RefreshMsg)
- [x] Define View type with constants for all 7 views
- [x] Implement View.String() method
- [x] Implement ParseView() function
- [x] `go build ./main/tui/...` passes
- [x] `go test ./...` passes
