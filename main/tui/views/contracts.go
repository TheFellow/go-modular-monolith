package views

import shared "github.com/TheFellow/go-modular-monolith/app/surfaces/tui/views"

type ViewModel = shared.ViewModel
type Interaction = shared.Interaction
type NavigateMsg = shared.NavigateMsg
type BackMsg = shared.BackMsg
type ErrorMsg = shared.ErrorMsg
type RefreshMsg = shared.RefreshMsg
type View = shared.View

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
