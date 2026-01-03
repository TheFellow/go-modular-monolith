package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/inventory/internal/dao"
)

type Queries struct {
	dao *dao.FileStockDAO
}

func New(stockDataPath string) (*Queries, error) {
	d := dao.NewFileStockDAO(stockDataPath)
	if err := d.Load(context.Background()); err != nil {
		return nil, err
	}
	return NewWithDAO(d), nil
}

func NewWithDAO(d *dao.FileStockDAO) *Queries {
	return &Queries{dao: d}
}
