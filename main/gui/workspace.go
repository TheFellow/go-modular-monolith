package main

// workspace identifies one top-level desktop destination. Keeping this closed
// vocabulary typed prevents route wiring, authorization probes, and dashboard
// cards from drifting apart through string literals.
type workspace string

const (
	workspaceDashboard   workspace = "dashboard"
	workspaceDrinks      workspace = "drinks"
	workspaceIngredients workspace = "ingredients"
	workspaceInventory   workspace = "inventory"
	workspaceMenus       workspace = "menus"
	workspaceOrders      workspace = "orders"
	workspaceAudit       workspace = "audit"
	workspaceTags        workspace = "tags"
)

func (w workspace) routeID() string { return string(w) }
