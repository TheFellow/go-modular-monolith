package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/ingredients/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/queries"
)

type Module struct {
	queries *queries.Queries
	create  *commands.Create
	update  *commands.Update
}

func NewModule() *Module {
	return &Module{
		queries: queries.New(),
		create:  commands.NewCreate(),
		update:  commands.NewUpdate(),
	}
}
