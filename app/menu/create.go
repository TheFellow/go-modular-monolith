package menu

import (
	"github.com/TheFellow/go-modular-monolith/app/menu/authz"
	"github.com/TheFellow/go-modular-monolith/app/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (m *Module) Create(ctx *middleware.Context, menu models.Menu) (models.Menu, error) {
	resource := cedar.Entity{
		UID:        cedar.NewEntityUID(models.MenuEntityType, cedar.String("")),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}

	return middleware.RunCommand(ctx, authz.ActionCreate, resource, func(mctx *middleware.Context, menu models.Menu) (models.Menu, error) {
		return m.commands.Create(mctx, menu)
	}, menu)
}
