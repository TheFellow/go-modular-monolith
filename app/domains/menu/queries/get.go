package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	cedar "github.com/cedar-policy/cedar-go"
)

func (q *Queries) Get(ctx context.Context, id cedar.EntityUID) (*models.Menu, error) {
	return q.dao.Get(ctx, id)
}
