package audit_test

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	auditauthz "github.com/TheFellow/go-modular-monolith/app/domains/audit/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestActionProjectorUsesIndependentListAndGetCapabilities(t *testing.T) {
	t.Parallel()
	entry := &models.AuditEntry{ID: entity.NewAuditEntryID()}
	projector := audit.ActionProjector{Authorize: func(_ context.Context, _ cedar.EntityUID, action cedar.EntityUID, resource cedar.Entity) error {
		if action == auditauthz.ActionGet {
			testutil.Equals(t, resource.UID, entry.ID.EntityUID())
			return apperrors.Permissionf("no detail")
		}
		return nil
	}}
	states, err := projector.Project(context.Background(), authn.Owner(), entry)
	testutil.Ok(t, err)
	testutil.Equals(t, states, []actions.State{
		{ID: audit.ControlList, Visible: true, Enabled: true},
		{ID: audit.ControlView, Visible: false, Enabled: false},
	})
}

func TestActionProjectorWithoutSelectionProjectsOnlyList(t *testing.T) {
	t.Parallel()
	states, err := audit.NewActionProjector().Project(context.Background(), authn.Owner(), nil)
	testutil.Ok(t, err)
	testutil.Equals(t, states, []actions.State{{ID: audit.ControlList, Visible: true, Enabled: true}})
}

func TestActionProjectorSurfacesEvaluatorFailures(t *testing.T) {
	t.Parallel()
	want := stderrors.New("policy evaluator unavailable")
	projector := audit.ActionProjector{Authorize: func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error { return want }}
	_, err := projector.Project(context.Background(), authn.Owner(), nil)
	testutil.ErrorIf(t, !stderrors.Is(err, want), "projection error = %v, want %v", err, want)
}
