package dao

import (
	"context"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

type DAO struct {
	store *store.Store
	tags  tag.Repository
}

func New(s *store.Store, tags tag.Repository) *DAO { return &DAO{store: s, tags: tags} }

func Register(ctx context.Context, s *store.Store) {
	s.Register(ctx, DrinkRow{})
	if err := backfillLegacyStatuses(ctx, s); err != nil {
		panic(err)
	}
}

// backfillLegacyStatuses is an explicit schema migration for databases created
// before Drink lifecycle state was persisted. Conversion remains strict so
// future corrupt or unknown values cannot masquerade as active Drinks.
func backfillLegacyStatuses(ctx context.Context, s *store.Store) error {
	return s.Write(ctx, func(tx *bstore.Tx) error {
		rows, err := bstore.QueryTx[DrinkRow](tx).FilterEqual("Status", "").List()
		if err != nil {
			return err
		}
		for i := range rows {
			rows[i].Status = string(drinksmodels.StatusActive)
			if err := tx.Update(&rows[i]); err != nil {
				return err
			}
		}
		return nil
	})
}
