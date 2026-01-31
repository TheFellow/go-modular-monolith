package commands

import "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/internal/dao"

type Commands struct {
	dao *dao.DAO
}

func New() *Commands {
	return &Commands{dao: dao.New()}
}

func NewWithDAO(d *dao.DAO) *Commands {
	return &Commands{dao: d}
}
