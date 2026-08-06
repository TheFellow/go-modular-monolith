package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type DAO struct {
	store *store.Store
	tags  tag.Repository
}

func New(s *store.Store, tags tag.Repository) *DAO { return &DAO{store: s, tags: tags} }

func Register(ctx context.Context, s *store.Store) {
	s.Register(ctx, StockRow{}, ReservationRow{})
}
