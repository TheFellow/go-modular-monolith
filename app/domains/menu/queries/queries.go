package queries

import "github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"

type Queries struct {
	dao *dao.DAO
}

func New() *Queries {
	return &Queries{dao: dao.New()}
}
