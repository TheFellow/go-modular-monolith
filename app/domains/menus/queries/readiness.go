package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) Readiness(ctx store.Context, id entity.MenuID) (*models.Menu, models.ReadinessReport, error) {
	menu, err := q.dao.Get(ctx, id)
	if err != nil {
		return nil, models.ReadinessReport{}, err
	}
	report, err := q.availability.Readiness(ctx, menu)
	return menu, report, err
}
