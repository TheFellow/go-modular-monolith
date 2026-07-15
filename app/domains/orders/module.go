package orders

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/queries"
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
