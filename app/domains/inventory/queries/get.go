package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

func (q *Queries) Get(ctx store.Context, ingredientID cedar.EntityUID) (*models.Inventory, error) {
	return q.dao.Get(ctx, ingredientID)
}
