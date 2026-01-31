package queries

import (
	inventorydao "github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) List(ctx store.Context, filter inventorydao.ListFilter) ([]*models.Inventory, error) {
	return q.dao.List(ctx, filter)
}
