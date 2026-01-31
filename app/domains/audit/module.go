package audit

import "github.com/TheFellow/go-modular-monolith/app/domains/audit/queries"

type Module struct {
	queries *queries.Queries
}

func NewModule() *Module {
	return &Module{
		queries: queries.New(),
	}
}
