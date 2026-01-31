package tui

import "github.com/TheFellow/go-modular-monolith/main/tui/views"

// NavigateMsg requests navigation to a different view.
type NavigateMsg = views.NavigateMsg

// BackMsg requests navigation to the previous view.
type BackMsg = views.BackMsg

// ErrorMsg carries an error to display in the status bar.
type ErrorMsg = views.ErrorMsg

// RefreshMsg requests the current view to reload its data.
type RefreshMsg = views.RefreshMsg

// View represents a navigable view in the TUI.
type View = views.View

const (
	ViewDashboard   = views.ViewDashboard
	ViewDrinks      = views.ViewDrinks
	ViewIngredients = views.ViewIngredients
	ViewInventory   = views.ViewInventory
	ViewMenus       = views.ViewMenus
	ViewOrders      = views.ViewOrders
	ViewAudit       = views.ViewAudit
)

// ParseView converts a string argument to a View.
func ParseView(s string) (View, bool) {
	return views.ParseView(s)
}
