package dao

import "github.com/TheFellow/go-modular-monolith/pkg/store"

type DAO struct{}

func New() *DAO { return &DAO{} }

func Register(s *store.Store) {
	s.Register(AuditEntryRow{})
}
