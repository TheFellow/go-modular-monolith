package commands

import (
	ingredientsqueries "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/queries"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type Commands struct {
	dao         *dao.DAO
	ingredients *ingredientsqueries.Queries
}

func New(s *store.Store) *Commands {
	return &Commands{
		dao:         dao.New(s),
		ingredients: ingredientsqueries.New(s),
	}
}
