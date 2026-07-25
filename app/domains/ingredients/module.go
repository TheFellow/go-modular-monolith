package ingredients

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/queries"
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type Module struct {
	queries  *queries.Queries
	commands *commands.Commands
	pipeline *middleware.Pipeline
}

func NewModule(ctx context.Context, s *store.Store, tags *tagging.Repository, pipeline *middleware.Pipeline) *Module {
	dao.Register(ctx, s)
	return &Module{
		queries:  queries.New(s, tags),
		commands: commands.New(s, tags),
		pipeline: pipeline,
	}
}
