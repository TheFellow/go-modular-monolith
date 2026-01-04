package commands

import "github.com/TheFellow/go-modular-monolith/app/ingredients/internal/dao"

type Commands struct {
	dao *dao.FileIngredientDAO
}

func New() *Commands {
	return &Commands{dao: dao.New()}
}

func NewWithDAO(d *dao.FileIngredientDAO) *Commands {
	return &Commands{dao: d}
}
