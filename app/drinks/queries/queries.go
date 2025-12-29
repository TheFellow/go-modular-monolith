package queries

import "github.com/TheFellow/go-modular-monolith/app/drinks/internal/dao"

type Queries struct {
	dao *dao.FileDrinkDAO
}

func New(dao *dao.FileDrinkDAO) *Queries {
	return &Queries{dao: dao}
}
