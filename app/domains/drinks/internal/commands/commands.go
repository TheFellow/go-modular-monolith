package commands

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	ingredientsqueries "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/queries"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type Commands struct {
	dao         *dao.DAO
	ingredients *ingredientsqueries.Queries
}

func New(s *store.Store, tags tag.Repository) *Commands {
	return &Commands{
		dao:         dao.New(s, tags),
		ingredients: ingredientsqueries.New(s, tags),
	}
}
