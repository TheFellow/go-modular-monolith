package commands

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	ingredientsqueries "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/queries"
	cedar "github.com/cedar-policy/cedar-go"
)

type Commands struct {
	dao         *dao.DAO
	ingredients ingredientReader
}

type ingredientReader interface {
	Get(ctx context.Context, id cedar.EntityUID) (ingredientsmodels.Ingredient, error)
}

func New() *Commands {
	return &Commands{
		dao:         dao.New(),
		ingredients: ingredientsqueries.New(),
	}
}

func NewWithDependencies(d *dao.DAO, ingredients ingredientReader) *Commands {
	return &Commands{dao: d, ingredients: ingredients}
}
