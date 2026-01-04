package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/drinks/queries"
)

type Module struct {
	queries      *queries.Queries
	create       *commands.Create
	updateRecipe *commands.UpdateRecipe
}

func NewModule() *Module {
	return &Module{
		queries:      queries.New(),
		create:       commands.NewCreate(),
		updateRecipe: commands.NewUpdateRecipe(),
	}
}
