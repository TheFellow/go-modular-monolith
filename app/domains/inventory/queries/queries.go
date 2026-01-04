package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
)

type Queries struct {
	dao *dao.FileStockDAO
}

func New() *Queries {
	return &Queries{dao: dao.New()}
}

func NewWithDAO(d *dao.FileStockDAO) *Queries {
	return &Queries{dao: d}
}
