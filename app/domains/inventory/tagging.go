package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

func (m *Module) registerTagTarget(targets *tagging.Registry) {
	targets.Register(tagging.Target{
		Type: entity.TypeInventory, GetAction: authz.ActionGet, TagAction: authz.ActionTag, UntagAction: authz.ActionUntag,
		Load: func(ctx store.Context, raw cedar.String) (tagging.TargetState, error) {
			id, err := entity.ParseInventoryID(string(raw))
			if err != nil {
				return tagging.TargetState{}, err
			}
			value, err := m.queries.GetByID(ctx, id)
			if err != nil {
				return tagging.TargetState{}, err
			}
			return tagging.TargetState{Entity: value.CedarEntity(), Tags: value.Tags}, nil
		},
	})
}
