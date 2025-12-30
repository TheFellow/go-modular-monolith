package drinks

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/drinks/queries"
)

type Module struct {
	queries *queries.Queries
	create  *commands.Create
}

func NewModule(drinksDataPath string) (*Module, error) {
	d := dao.NewFileDrinkDAO(drinksDataPath)
	if err := d.Load(context.Background()); err != nil {
		return nil, err
	}

	return &Module{
		queries: queries.NewWithDAO(d),
		create:  commands.NewCreate(d),
	}, nil
}
