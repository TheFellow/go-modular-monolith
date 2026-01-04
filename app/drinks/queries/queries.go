package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/dao"
)

type Queries struct {
	dao *dao.FileDrinkDAO
}

func New() *Queries {
	return &Queries{dao: dao.New()}
}

func NewWithDAO(d *dao.FileDrinkDAO) *Queries {
	return &Queries{dao: d}
}
