package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/cedar-policy/cedar-go"
)

func (m *Module) Create(ctx *middleware.Context, drink models.Drink) (models.Drink, error) {
	resource := cedar.Entity{
		UID:        cedar.NewEntityUID(cedar.EntityType("Mixology::Drink"), cedar.String("")),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}

	return middleware.RunCommand(ctx, authz.ActionCreate, resource, func(mctx *middleware.Context, drink models.Drink) (models.Drink, error) {
		d, err := m.commands.Create(mctx, drink)
		if err != nil {
			return models.Drink{}, err
		}
		return d, nil
	}, drink)
}
