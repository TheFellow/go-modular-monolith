package audit

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/queries"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type Module struct {
	queries  *queries.Queries
	pipeline *middleware.Pipeline
}

func NewModule(s *store.Store, pipeline *middleware.Pipeline) *Module {
	return &Module{
		queries:  queries.New(s),
		pipeline: pipeline,
	}
}
