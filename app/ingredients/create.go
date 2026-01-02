package ingredients

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/cedar-policy/cedar-go"
)

type CreateRequest struct {
	Name        string
	Category    models.Category
	Unit        models.Unit
	Description string
}

type CreateResponse struct {
	Ingredient models.Ingredient
}

func (m *Module) Create(ctx context.Context, req CreateRequest) (CreateResponse, error) {
	resource := cedar.Entity{
		UID:        cedar.NewEntityUID(cedar.EntityType("Mixology::Ingredient"), cedar.String("")),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}

	return middleware.RunCommand(ctx, authz.ActionCreate, resource, func(mctx *middleware.Context, req CreateRequest) (CreateResponse, error) {
		cmdReq := commands.CreateRequest{
			Name:        req.Name,
			Category:    req.Category,
			Unit:        req.Unit,
			Description: req.Description,
		}
		i, err := m.create.Execute(mctx, cmdReq)
		if err != nil {
			return CreateResponse{}, err
		}
		return CreateResponse{Ingredient: i}, nil
	}, req)
}
