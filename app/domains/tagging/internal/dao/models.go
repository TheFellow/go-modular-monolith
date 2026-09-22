package dao

type entityTagRow struct {
	ID         uint64
	Revision   uint64 `json:"-" store:"revision"`
	EntityType string `store:"unique=EntityType+EntityID+Key"`
	EntityID   string
	Key        string `store:"index=Key+Value+EntityType+EntityID"`
	Value      string
}

func (entityTagRow) StoreModelName() string { return "entity_tags" }
