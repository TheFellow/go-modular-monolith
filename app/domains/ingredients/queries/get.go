package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

func (q *Queries) Get(ctx store.Context, id cedar.EntityUID) (*models.Ingredient, error) {
	return q.dao.Get(ctx, id)
}
