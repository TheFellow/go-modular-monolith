package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/ingredients/internal/dao"
)

type Queries struct {
	dao *dao.FileIngredientDAO
}

func New(ingredientsDataPath string) (*Queries, error) {
	d := dao.NewFileIngredientDAO(ingredientsDataPath)
	if err := d.Load(context.Background()); err != nil {
		return nil, err
	}
	return NewWithDAO(d), nil
}

func NewWithDAO(d *dao.FileIngredientDAO) *Queries {
	return &Queries{dao: d}
}
