package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/availability"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type Queries struct {
	dao          *dao.DAO
	availability *availability.AvailabilityCalculator
}

func New(s *store.Store) *Queries {
	return &Queries{dao: dao.New(s), availability: availability.New(s)}
}
