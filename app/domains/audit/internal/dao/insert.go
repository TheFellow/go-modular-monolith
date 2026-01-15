package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) Insert(ctx context.Context, entry models.AuditEntry) error {
	return d.write(ctx, func(tx *bstore.Tx) error {
		row := toRow(entry)
		return store.MapError(tx.Insert(&row), "insert audit entry %q", row.ID)
	})
}

