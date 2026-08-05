package audit

import (
	"context"

	auditauthz "github.com/TheFellow/go-modular-monolith/app/domains/audit/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	pkgAuthz "github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	cedar "github.com/cedar-policy/cedar-go"
)

// Stable capability identities shared by audit presentation adapters.
const (
	ControlList actions.ID = "audit.list"
	ControlView actions.ID = "audit.view"
)

// ActionProjector produces framework-neutral audit capability state.
type ActionProjector struct{ Authorize pkgAuthz.EntityAuthorizer }

func NewActionProjector() ActionProjector {
	return ActionProjector{Authorize: pkgAuthz.AuthorizeEntity}
}

// Project always projects collection reads and additionally projects the
// selected entry's detail read. Filtering, paging, and navigation are local UI
// mechanics rather than separate domain capabilities.
func (p ActionProjector) Project(ctx context.Context, principal cedar.EntityUID, selected *models.AuditEntry) ([]actions.State, error) {
	authorize := p.Authorize
	if authorize == nil {
		authorize = NewActionProjector().Authorize
	}
	permission := func(action cedar.EntityUID, resource cedar.Entity) actions.Permission {
		return actions.Require(func(ctx context.Context) error { return authorize(ctx, principal, action, resource) })
	}
	declaration := actions.Group{Controls: []actions.Control{{
		ID:         ControlList,
		Permission: permission(auditauthz.ActionList, auditauthz.AuditEntry{UID: cedar.NewEntityUID(auditauthz.AuditEntryType, "workspace")}.CedarEntity()),
	}}}
	if selected != nil {
		declaration.Controls = append(declaration.Controls, actions.Control{
			ID: ControlView, Permission: permission(auditauthz.ActionGet, selected.CedarEntity()),
		})
	}
	return actions.Evaluate(ctx, declaration)
}
