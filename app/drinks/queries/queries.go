package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/dao"
)

type Queries struct {
	dao *dao.FileDrinkDAO
}

func New(drinksDataPath string) (*Queries, error) {
	d := dao.NewFileDrinkDAO(drinksDataPath)
	if err := d.Load(context.Background()); err != nil {
		return nil, err
	}
	return NewWithDAO(d), nil
}

func NewWithDAO(d *dao.FileDrinkDAO) *Queries {
	return &Queries{dao: d}
}
