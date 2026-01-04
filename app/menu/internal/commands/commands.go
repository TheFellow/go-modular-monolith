package commands

import (
	drinksq "github.com/TheFellow/go-modular-monolith/app/drinks/queries"
	"github.com/TheFellow/go-modular-monolith/app/menu/internal/availability"
	"github.com/TheFellow/go-modular-monolith/app/menu/internal/dao"
)

type Commands struct {
	dao          *dao.FileMenuDAO
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
