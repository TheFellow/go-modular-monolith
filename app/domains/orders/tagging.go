package orders

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

func (m *Module) registerTagTarget(targets *tagging.Registry) {
	targets.Register(tagging.Target{
		Type: entity.TypeOrder, GetAction: authz.ActionGet, TagAction: authz.ActionTag, UntagAction: authz.ActionUntag,
		Active: m.queries.ActiveIDs,
		Load: func(ctx store.Context, raw cedar.String) (tagging.TargetState, error) {
			id, err := entity.ParseOrderID(string(raw))
			if err != nil {
				return tagging.TargetState{}, err
			}
			value, err := m.queries.Get(ctx, id)
			if err != nil {
				return tagging.TargetState{}, err
			}
			menu, err := m.menus.Get(ctx, value.MenuID)
			if err != nil {
				return tagging.TargetState{}, err
			}
			return tagging.TargetState{Entity: value.CedarEntity(), DisplayName: fmt.Sprintf("Order for %s", menu.Name), Tags: value.Tags}, nil
		},
	})
}
