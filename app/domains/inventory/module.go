package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/queries"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type Module struct {
	queries  *queries.Queries
	commands *commands.Commands
	pipeline *middleware.Pipeline
}

func NewModule(s *store.Store, pipeline *middleware.Pipeline) *Module {
	dao.Register(s)
	return &Module{
		queries:  queries.New(),
		commands: commands.New(),
		pipeline: pipeline,
	}
}
