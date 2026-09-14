package orders

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	pkgauthz "github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type amendmentSelection []*models.Order

func (s amendmentSelection) CedarEntity() cedar.Entity { return s[0].CedarEntity() }

// AmendBatch is one Orders command. The complete selection is authorized and
// revision-checked before planning; one event describes all reservation changes.
func (m *Module) AmendBatch(ctx *middleware.Context, requests []models.Amendment) ([]*models.Order, error) {
	if len(requests) == 0 {
		return nil, errors.Invalidf("amendment selection is required")
	}
	result, err := m.pipeline.LoadCommand(ctx, authz.ActionAmend,
		func(ctx *middleware.Context) (amendmentSelection, error) {
			selected := amendmentSelection{}
			seen := map[entity.OrderID]bool{}
			for _, request := range requests {
				if seen[request.OrderID] {
					return nil, errors.Invalidf("duplicate order in amendment selection")
				}
				seen[request.OrderID] = true
				order, err := m.queries.Get(ctx, request.OrderID)
				if err != nil {
					return nil, err
				}
				if request.Revision == 0 || request.Revision != order.Revision {
					return nil, errors.Conflictf("order %s changed; reload selection", request.OrderID.String())
				}
				if err := pkgauthz.AuthorizeWithEntity(ctx.Principal(), authz.ActionAmend, order.CedarEntity()); err != nil {
					return nil, err
				}
				ctx.ReferenceEntity(order.ID.EntityUID())
				selected = append(selected, order)
			}
			return selected, nil
		},
		func(ctx *middleware.Context, selected amendmentSelection) (amendmentSelection, error) {
			orders, err := m.commands.AmendBatch(ctx, selected, requests)
			if err != nil {
				return nil, err
			}
			for _, order := range orders {
				if err := pkgauthz.AuthorizeWithEntity(ctx.Principal(), authz.ActionAmend, order.CedarEntity()); err != nil {
					return nil, err
				}
			}
			return amendmentSelection(orders), nil
		})
	if err != nil {
		return nil, err
	}
	return result, nil
}
