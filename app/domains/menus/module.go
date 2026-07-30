package menus

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/queries"
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type Module struct {
	queries   *queries.Queries
	analytics *queries.AnalyticsCalculator
	commands  *commands.Commands
	pipeline  *middleware.Pipeline
}

func NewModule(ctx context.Context, s *store.Store, tags tag.Repository, targets *tagging.Registry, pipeline *middleware.Pipeline) *Module {
	dao.Register(ctx, s)
	m := &Module{
		queries:   queries.New(s, tags),
		analytics: queries.NewAnalyticsCalculator(s, tags),
		commands:  commands.New(s, tags),
		pipeline:  pipeline,
	}
	m.registerTagTarget(targets)
	return m
}
