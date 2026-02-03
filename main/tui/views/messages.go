package views

// NavigateMsg requests navigation to a different view.
type NavigateMsg struct {
	To View
}

// BackMsg requests navigation to the previous view.
type BackMsg struct{}

// ErrorMsg carries an error to display in the status bar.
type ErrorMsg struct {
	Err error
}

// RefreshMsg requests the current view to reload its data.
type RefreshMsg struct{}

// View represents a navigable view in the TUI.
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

// String returns the display name for the view.
func (v View) String() string {
	switch v {
	case ViewDashboard:
		return "dashboard"
	case ViewDrinks:
		return "drinks"
	case ViewIngredients:
		return "ingredients"
	case ViewInventory:
		return "inventory"
	case ViewMenus:
		return "menus"
	case ViewOrders:
		return "orders"
	case ViewAudit:
		return "audit"
	default:
		return "unknown"
	}
}
