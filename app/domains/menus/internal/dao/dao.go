package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type DAO struct{ store *store.Store }

func New(s *store.Store) *DAO { return &DAO{store: s} }

func Register(ctx context.Context, s *store.Store) {
	s.Register(ctx, MenuRow{})
}
