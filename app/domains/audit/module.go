package audit

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/queries"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type Module struct {
	dao      *dao.DAO
	queries  *queries.Queries
	pipeline *middleware.Pipeline
}

func NewModule(ctx context.Context, s *store.Store, pipeline *middleware.Pipeline) *Module {
	dao.Register(ctx, s)
	return &Module{
		dao:      dao.New(s),
		queries:  queries.New(s),
		pipeline: pipeline,
	}
}
