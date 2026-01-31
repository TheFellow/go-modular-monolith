//go:generate go run ./gen

package entity

// EntityDef defines an entity type for code generation.
type EntityDef struct {
	Name   string // e.g., "Drink"
	Type   string // e.g., "Mixology::Drink"
	Prefix string // e.g., "drk"
}

// Entities defines all entity types in the system.
var Entities = []EntityDef{
	{Name: "Drink", Type: "Mixology::Drink", Prefix: "drk"},
	{Name: "Ingredient", Type: "Mixology::Ingredient", Prefix: "ing"},
	{Name: "Menu", Type: "Mixology::Menu", Prefix: "mnu"},
	{Name: "Order", Type: "Mixology::Order", Prefix: "ord"},
	{Name: "Inventory", Type: "Mixology::Inventory", Prefix: "inv"},
	{Name: "AuditEntry", Type: "Mixology::AuditEntry", Prefix: "aud"},
}
