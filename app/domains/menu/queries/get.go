package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/dao"
	cedar "github.com/cedar-policy/cedar-go"
)

func (q *Queries) Get(ctx dao.Context, id cedar.EntityUID) (*models.Menu, error) {
	return q.dao.Get(ctx, id)
}
