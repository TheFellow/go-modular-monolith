package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/internal/dao"
)

type Queries struct {
	dao *dao.FileIngredientDAO
}

func New() *Queries {
	return &Queries{dao: dao.New()}
}

func NewWithDAO(d *dao.FileIngredientDAO) *Queries {
	return &Queries{dao: d}
}
