package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Create(ctx *middleware.Context, drink models.Drink) (*models.Drink, error) {
	return middleware.RunCommand(ctx, authz.ActionCreate,
		func(*middleware.Context) (*models.Drink, error) {
			toCreate := drink
			if toCreate.ID.Type == "" {
				toCreate.ID = models.NewDrinkID(string(toCreate.ID.ID))
			}
			return &toCreate, nil
		},
		m.commands.Create,
	)
}
