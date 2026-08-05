package inventory

import (
	"context"

	ingredientsqueries "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/queries"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/queries"
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type Module struct {
	queries     *queries.Queries
	ingredients *ingredientsqueries.Queries
	commands    *commands.Commands
	pipeline    *middleware.Pipeline
}

func NewModule(ctx context.Context, s *store.Store, tags tag.Repository, targets *tagging.Registry, pipeline *middleware.Pipeline) *Module {
	dao.Register(ctx, s)
	m := &Module{
		queries:     queries.New(s, tags),
		ingredients: ingredientsqueries.New(s, tags),
		commands:    commands.New(s, tags),
		pipeline:    pipeline,
	}
	m.registerTagTarget(targets)
	return m
}
