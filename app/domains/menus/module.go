package menus

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/queries"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type Module struct {
	queries  *queries.Queries
	commands *commands.Commands
	pipeline *middleware.Pipeline
}

func NewModule(ctx context.Context, s *store.Store, pipeline *middleware.Pipeline) *Module {
	dao.Register(ctx, s)
	return &Module{
		queries:  queries.New(s),
		commands: commands.New(s),
		pipeline: pipeline,
	}
}
