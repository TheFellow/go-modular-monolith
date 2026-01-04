package menu

import (
	"github.com/TheFellow/go-modular-monolith/app/menu/authz"
	"github.com/TheFellow/go-modular-monolith/app/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (m *Module) RemoveDrink(ctx *middleware.Context, menuID cedar.EntityUID, drinkID cedar.EntityUID) (models.Menu, error) {
	resource := cedar.Entity{
		UID:        menuID,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}

	return middleware.RunCommand(ctx, authz.ActionRemoveDrink, resource, func(mctx *middleware.Context, _ struct{}) (models.Menu, error) {
		return m.commands.RemoveDrink(mctx, menuID, drinkID)
	}, struct{}{})
}
