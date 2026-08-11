package tagging

type entityTagRow struct {
	ID         uint64
	Revision   uint64 `json:"-" store:"revision"`
	EntityType string `store:"unique=EntityType+EntityID+Key"`
	EntityID   string
	Key        string `store:"index"`
	Value      string
}

// Reference identifies an active entity carrying a tag.
type Reference struct {
	EntityType string `json:"entity_type" table:"ENTITY TYPE"`
	EntityName string `json:"entity_name" table:"ENTITY NAME"`
	EntityID   string `json:"entity_id" table:"ENTITY ID"`
	Tag        string `json:"tag" table:"TAG"`
}

// Summary reports active tag usage across each supported entity type.
type Summary struct {
	Tag         string `json:"tag" table:"TAG"`
	Total       int    `json:"total" table:"TOTAL"`
	Drinks      int    `json:"drinks" table:"DRINKS"`
	Ingredients int    `json:"ingredients" table:"INGREDIENTS"`
	Inventory   int    `json:"inventory" table:"INVENTORY"`
	Menus       int    `json:"menus" table:"MENUS"`
	Orders      int    `json:"orders" table:"ORDERS"`
}
