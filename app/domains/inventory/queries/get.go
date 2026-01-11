package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	cedar "github.com/cedar-policy/cedar-go"
)

func (q *Queries) Get(ctx context.Context, ingredientID cedar.EntityUID) (*models.Inventory, error) {
	return q.dao.Get(ctx, ingredientID)
}
