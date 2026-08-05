package tagging

import (
	"context"

	taggingauthz "github.com/TheFellow/go-modular-monolith/app/domains/tagging/authz"
	pkgAuthz "github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	cedar "github.com/cedar-policy/cedar-go"
)

// Stable identities shared by tagging presentation adapters.
const (
	ControlInspect actions.ID = "tagging.inspect"
	ControlTag     actions.ID = "tagging.tag"
	ControlUntag   actions.ID = "tagging.untag"
	ControlShow    actions.ID = "tagging.show"
	ControlSummary actions.ID = "tagging.summary"
)

// ActionProjector projects both tagging-owned discovery operations and the
// owning domain's target-specific tag operations. The registry is what keeps
// this cross-entity domain from guessing another domain's Cedar actions.
type ActionProjector struct {
	Authorize pkgAuthz.EntityAuthorizer
	registry  *Registry
}

func NewActionProjector(registry *Registry) ActionProjector {
	return ActionProjector{Authorize: pkgAuthz.AuthorizeEntity, registry: registry}
}

func (m *Module) NewActionProjector() ActionProjector {
	return NewActionProjector(m.registry)
}

// ProjectDiscovery returns the two capabilities owned by tagging itself.
func (p ActionProjector) ProjectDiscovery(ctx context.Context, principal cedar.EntityUID) ([]actions.State, error) {
	resource := func(id cedar.String) cedar.Entity {
		return taggingauthz.TagDiscovery{UID: cedar.NewEntityUID(taggingauthz.TagDiscoveryType, id)}.CedarEntity()
	}
	return actions.Evaluate(ctx, actions.Group{Controls: []actions.Control{
		{ID: ControlShow, Permission: p.permission(principal, taggingauthz.ActionShow, resource("show"))},
		{ID: ControlSummary, Permission: p.permission(principal, taggingauthz.ActionSummary, resource("summary"))},
	}})
}

// ProjectTarget resolves the actions registered by the target's owning domain
// and evaluates them against the target's complete Cedar entity.
func (p ActionProjector) ProjectTarget(ctx context.Context, principal cedar.EntityUID, target cedar.Entity) ([]actions.State, error) {
	registration, err := p.registry.resolve(target.UID.Type)
	if err != nil {
		return nil, err
	}
	return actions.Evaluate(ctx, actions.Group{Controls: []actions.Control{
		{ID: ControlInspect, Permission: p.permission(principal, registration.GetAction, target)},
		{ID: ControlTag, Permission: p.permission(principal, registration.TagAction, target)},
		{ID: ControlUntag, Permission: p.permission(principal, registration.UntagAction, target)},
	}})
}

func (p ActionProjector) permission(principal, action cedar.EntityUID, resource cedar.Entity) actions.Permission {
	authorize := p.Authorize
	if authorize == nil {
		authorize = pkgAuthz.AuthorizeEntity
	}
	return actions.Require(func(ctx context.Context) error { return authorize(ctx, principal, action, resource) })
}
