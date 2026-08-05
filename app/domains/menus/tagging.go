package menus

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

func (m *Module) registerTagTarget(targets *tagging.Registry) {
	targets.Register(tagging.Target{
		Type: entity.TypeMenu, GetAction: authz.ActionGet, TagAction: authz.ActionTag, UntagAction: authz.ActionUntag,
		Active: m.queries.ActiveIDs,
		Load: func(ctx store.Context, raw cedar.String) (tagging.TargetState, error) {
			id, err := entity.ParseMenuID(string(raw))
			if err != nil {
				return tagging.TargetState{}, err
			}
			value, err := m.queries.Get(ctx, id)
			if err != nil {
				return tagging.TargetState{}, err
			}
			return tagging.TargetState{Entity: value.CedarEntity(), DisplayName: value.Name, Tags: value.Tags}, nil
		},
	})
}
