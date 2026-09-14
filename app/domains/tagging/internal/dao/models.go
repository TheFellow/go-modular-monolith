package dao

type entityTagRow struct {
	ID         uint64
	Revision   uint64 `json:"-" store:"revision"`
	EntityType string `store:"unique=EntityType+EntityID+Key"`
	EntityID   string
	Key        string `store:"index"`
	Value      string
}

// StoreModelName preserves the identity used before persistence moved into dao.
func (entityTagRow) StoreModelName() string {
	return "github.com/TheFellow/go-modular-monolith/app/domains/tagging.entityTagRow"
}
