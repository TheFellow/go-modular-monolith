package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

// GetByID is the private cross-cutting lookup used when a caller has an
// inventory entity UID rather than its ingredient lookup key.
func (q *Queries) GetByID(ctx store.Context, id entity.InventoryID) (*models.Inventory, error) {
	return q.dao.GetByID(ctx, id)
}
