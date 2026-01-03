package drinks

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/drinks/queries"
	"github.com/TheFellow/go-modular-monolith/app/ingredients"
)

type Module struct {
	queries      *queries.Queries
	create       *commands.Create
	updateRecipe *commands.UpdateRecipe
}

func NewModule(drinksDataPath string, ingredients *ingredients.Module) (*Module, error) {
	d := dao.NewFileDrinkDAO(drinksDataPath)
	if err := d.Load(context.Background()); err != nil {
		return nil, err
	}

	return &Module{
		queries:      queries.NewWithDAO(d),
		create:       commands.NewCreate(d, ingredients),
		updateRecipe: commands.NewUpdateRecipe(d, ingredients),
	}, nil
}
