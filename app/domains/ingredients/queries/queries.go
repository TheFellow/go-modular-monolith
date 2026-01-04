package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/internal/dao"
)

type Queries struct {
	dao *dao.DAO
}

func New() *Queries {
	return &Queries{dao: dao.New()}
}

func NewWithDAO(d *dao.DAO) *Queries {
	return &Queries{dao: d}
}
