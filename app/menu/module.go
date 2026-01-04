package menu

import (
	"github.com/TheFellow/go-modular-monolith/app/menu/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/menu/queries"
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
