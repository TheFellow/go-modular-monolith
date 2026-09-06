package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) Readiness(ctx store.Context, id entity.MenuID) (*models.Menu, models.ReadinessReport, error) {
	var menu *models.Menu
	var report models.ReadinessReport
	err := q.store.ReadContext(ctx, func(tx *store.Tx) error {
		snapshot := readSnapshot{Context: ctx, tx: tx}
		var err error
		menu, err = q.dao.Get(snapshot, id)
		if err != nil {
			return err
		}
		report, err = q.availability.Readiness(snapshot, menu)
		return err
	})
	return menu, report, err
}

type readSnapshot struct {
	store.Context
	tx *store.Tx
}

func (c readSnapshot) Transaction() (*store.Tx, bool) { return c.tx, true }
