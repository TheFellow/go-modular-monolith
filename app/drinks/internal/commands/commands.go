package commands

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/dao"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	ingredientsqueries "github.com/TheFellow/go-modular-monolith/app/ingredients/queries"
	cedar "github.com/cedar-policy/cedar-go"
)

type Commands struct {
	dao         *dao.FileDrinkDAO
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

func NewWithDependencies(d *dao.FileDrinkDAO, ingredients ingredientReader) *Commands {
	return &Commands{dao: d, ingredients: ingredients}
}
