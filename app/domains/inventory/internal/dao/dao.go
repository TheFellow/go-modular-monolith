package dao

import "github.com/TheFellow/go-modular-monolith/pkg/store"

type DAO struct{}

func New() *DAO { return &DAO{} }

func init() {
	store.RegisterTypes(StockRow{})
}
