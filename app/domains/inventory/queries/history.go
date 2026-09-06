package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) History(ctx store.Context, id entity.InventoryID) ([]models.Movement, error) {
	return q.dao.History(ctx, id)
}
