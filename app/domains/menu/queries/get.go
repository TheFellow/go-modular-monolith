package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

func (q *Queries) Get(ctx context.Context, id cedar.EntityUID) (models.Menu, error) {
	menu, ok, err := q.dao.Get(ctx, string(id.ID))
	if err != nil {
		return models.Menu{}, errors.Internalf("get menu %s: %w", id.ID, err)
	}
	if !ok {
		return models.Menu{}, errors.NotFoundf("menu %s not found", id.ID)
	}
	return menu, nil
}
