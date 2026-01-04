package menu

import (
	"github.com/TheFellow/go-modular-monolith/app/menu/authz"
	menucommands "github.com/TheFellow/go-modular-monolith/app/menu/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (m *Module) AddDrink(ctx *middleware.Context, menuID cedar.EntityUID, drinkID cedar.EntityUID) (models.Menu, error) {
	return middleware.RunCommand(ctx, authz.ActionAddDrink, m.commands.AddDrink, menucommands.AddDrinkParams{
		MenuID:  menuID,
		DrinkID: drinkID,
	})
}
