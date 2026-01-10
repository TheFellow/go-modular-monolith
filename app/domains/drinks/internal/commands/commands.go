package commands

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	ingredientsqueries "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/queries"
)

type Commands struct {
	dao         *dao.DAO
	ingredients *ingredientsqueries.Queries
}

func New() *Commands {
	return &Commands{
		dao:         dao.New(),
		ingredients: ingredientsqueries.New(),
	}
}
