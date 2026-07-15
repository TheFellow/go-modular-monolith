package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
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
		queries:  queries.New(s),
		commands: commands.New(s),
		pipeline: pipeline,
	}
}
