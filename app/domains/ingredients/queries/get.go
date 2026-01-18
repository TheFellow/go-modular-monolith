package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/dao"
	cedar "github.com/cedar-policy/cedar-go"
)

func (q *Queries) Get(ctx dao.Context, id cedar.EntityUID) (*models.Ingredient, error) {
	return q.dao.Get(ctx, id)
}
