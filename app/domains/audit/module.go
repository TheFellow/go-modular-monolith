package audit

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/queries"
)

type Module struct {
	dao     *dao.DAO
	queries *queries.Queries
}

func NewModule() *Module {
	return &Module{
		dao:     dao.New(),
		queries: queries.New(),
	}
}
