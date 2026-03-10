package commands

import (
	drinksq "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/availability"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/dao"
)

type Commands struct {
	dao          *dao.DAO
	availability *availability.AvailabilityCalculator
	drinks       *drinksq.Queries
}

func New() *Commands {
	return &Commands{
		dao:          dao.New(),
		availability: availability.New(),
		drinks:       drinksq.New(),
	}
}
