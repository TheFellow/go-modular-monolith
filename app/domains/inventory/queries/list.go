package queries

import (
	inventorydao "github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/dao"
)

func (q *Queries) List(ctx dao.Context, filter inventorydao.ListFilter) ([]*models.Inventory, error) {
	return q.dao.List(ctx, filter)
}
