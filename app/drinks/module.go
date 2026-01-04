package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/drinks/queries"
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
