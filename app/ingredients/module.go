package ingredients

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/ingredients/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/queries"
)

type Module struct {
	queries *queries.Queries
	create  *commands.Create
	update  *commands.Update
}

func NewModule(ingredientsDataPath string) (*Module, error) {
	d := dao.NewFileIngredientDAO(ingredientsDataPath)
	if err := d.Load(context.Background()); err != nil {
		return nil, err
	}

	return &Module{
		queries: queries.NewWithDAO(d),
		create:  commands.NewCreate(d),
		update:  commands.NewUpdate(d),
	}, nil
}
