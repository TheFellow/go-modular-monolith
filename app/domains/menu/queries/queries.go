package queries

import "github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"

type Queries struct {
	dao *dao.FileMenuDAO
}

func New() *Queries {
	return &Queries{dao: dao.New()}
}

func NewWithDAO(d *dao.FileMenuDAO) *Queries {
	return &Queries{dao: d}
}
