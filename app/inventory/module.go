package inventory

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/inventory/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/inventory/queries"
)

type Module struct {
	ingredients *ingredients.Module
	queries     *queries.Queries
	adjust      *commands.Adjust
	set         *commands.Set
}

func NewModule(stockDataPath string, ingredientsModule *ingredients.Module) (*Module, error) {
	d := dao.NewFileStockDAO(stockDataPath)
	if err := d.Load(context.Background()); err != nil {
		return nil, err
	}

	return &Module{
		ingredients: ingredientsModule,
		queries:     queries.NewWithDAO(d),
		adjust:      commands.NewAdjust(d),
		set:         commands.NewSet(d),
	}, nil
}
