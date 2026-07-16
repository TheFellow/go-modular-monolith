package commands

import (
	drinksq "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/availability"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type Commands struct {
	dao          *dao.DAO
	availability *availability.AvailabilityCalculator
	drinks       *drinksq.Queries
}

func New(s *store.Store) *Commands {
	return &Commands{
		dao:          dao.New(s),
		availability: availability.New(s),
		drinks:       drinksq.New(s),
	}
}
