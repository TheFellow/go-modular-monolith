package components

import "github.com/charmbracelet/lipgloss"

// EmptyState displays a message when there's no data.
type EmptyState struct {
	message string
	style   lipgloss.Style
}

func NewEmptyState(message string, style lipgloss.Style) EmptyState {
	return EmptyState{
		message: message,
		style:   style,
	}
}

func (e EmptyState) View() string {
	return e.style.Render(e.message)
}

const (
	EmptyDrinks      = "No drinks found"
	EmptyIngredients = "No ingredients found"
	EmptyInventory   = "No inventory items"
	EmptyMenus       = "No menus found"
	EmptyOrders      = "No orders found"
	EmptyAudit       = "No audit entries"
)
