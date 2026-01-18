package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) Insert(ctx store.Context, entry models.AuditEntry) error {
	return store.Write(ctx, func(tx *bstore.Tx) error {
		row := toRow(entry)
		return store.MapError(tx.Insert(&row), "insert audit entry %q", row.ID)
	})
}
