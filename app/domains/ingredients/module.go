package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/queries"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type Module struct {
	queries  *queries.Queries
	commands *commands.Commands
}

func NewModule(s *store.Store) *Module {
	dao.Register(s)
	return &Module{
		queries:  queries.New(),
		commands: commands.New(),
	}
}
