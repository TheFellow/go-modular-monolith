package queries

import "github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/dao"

type Queries struct {
	dao *dao.FileOrderDAO
}

func New() *Queries {
	return &Queries{dao: dao.New()}
}

func NewWithDAO(d *dao.FileOrderDAO) *Queries {
	return &Queries{dao: d}
}
