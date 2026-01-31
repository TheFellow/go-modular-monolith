package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/queries"
)

type Module struct {
	queries  *queries.Queries
	commands *commands.Commands
}

func NewModule() *Module {
	return &Module{
		queries:  queries.New(),
		commands: commands.New(),
	}
}
