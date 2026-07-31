package views

import shared "github.com/TheFellow/go-modular-monolith/app/surfaces/tui/views"

type (
	View        = shared.View
	NavigateMsg = shared.NavigateMsg
	BackMsg     = shared.BackMsg
	ErrorMsg    = shared.ErrorMsg
	RefreshMsg  = shared.RefreshMsg
	Interaction = shared.Interaction
	ViewModel   = shared.ViewModel
)

const (
	ViewDashboard   = shared.ViewDashboard
	ViewDrinks      = shared.ViewDrinks
	ViewIngredients = shared.ViewIngredients
	ViewInventory   = shared.ViewInventory
	ViewMenus       = shared.ViewMenus
	ViewOrders      = shared.ViewOrders
	ViewAudit       = shared.ViewAudit
	ViewTags        = shared.ViewTags
)
