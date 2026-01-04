package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/cedar-policy/cedar-go"
)

type UpdateRequest struct {
	ID          cedar.EntityUID
	Name        string
	Category    models.Category
	Unit        models.Unit
	Description string
}

type UpdateResponse struct {
	Ingredient models.Ingredient
}

func (m *Module) Update(ctx *middleware.Context, req UpdateRequest) (UpdateResponse, error) {
	resource := cedar.Entity{
		UID:        req.ID,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}

	return middleware.RunCommand(ctx, authz.ActionUpdate, resource, func(mctx *middleware.Context, req UpdateRequest) (UpdateResponse, error) {
		cmdReq := commands.UpdateRequest{
			ID:          string(req.ID.ID),
			Name:        req.Name,
			Category:    req.Category,
			Unit:        req.Unit,
			Description: req.Description,
		}
		i, err := m.commands.Update(mctx, cmdReq)
		if err != nil {
			return UpdateResponse{}, err
		}
		return UpdateResponse{Ingredient: i}, nil
	}, req)
}
