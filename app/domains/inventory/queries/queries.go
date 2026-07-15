package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type Queries struct {
	dao *dao.DAO
}

func New(s *store.Store) *Queries {
	return &Queries{dao: dao.New(s)}
}
