package audit

import (
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

func NewModule(s *store.Store, pipeline *middleware.Pipeline) *Module {
	dao.Register(s)
	return &Module{
		dao:      dao.New(),
		queries:  queries.New(),
		pipeline: pipeline,
	}
}
