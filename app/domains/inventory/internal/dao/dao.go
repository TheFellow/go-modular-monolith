package dao

import "github.com/TheFellow/go-modular-monolith/pkg/store"

type DAO struct{ store *store.Store }

func New(s *store.Store) *DAO { return &DAO{store: s} }

func Register(s *store.Store) {
	s.Register(StockRow{})
}
