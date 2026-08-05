//nolint:paralleltest // projector overrides are local to each view model.
package tui

import (
	"context"
	stderrors "errors"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	auditauthz "github.com/TheFellow/go-modular-monolith/app/domains/audit/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/charmbracelet/bubbles/list"
)

func TestGetDenialHidesAuditDetail(t *testing.T) {
	fix := testutil.NewFixture(t)
	vm := NewListViewModel(fix.App)
	entry := models.AuditEntry{ID: entity.NewAuditEntryID()}
	vm.shell.SetResult([]list.Item{newAuditItem(entry)}, nil)
	vm.projector = audit.ActionProjector{Authorize: func(_ context.Context, _ cedar.EntityUID, action cedar.EntityUID, _ cedar.Entity) error {
		if action == auditauthz.ActionGet {
			return apperrors.Permissionf("denied")
		}
		return nil
	}}
	vm.syncActions()
	vm.syncDetail()
	testutil.ErrorIf(t, !strings.Contains(vm.detail.View(), "Select an entry"), "denied detail was rendered: %s", vm.detail.View())
}

func TestProjectionEvaluatorFailureSurfaces(t *testing.T) {
	fix := testutil.NewFixture(t)
	want := stderrors.New("policy evaluator unavailable")
	vm := NewListViewModel(fix.App)
	vm.projector = audit.ActionProjector{Authorize: func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error { return want }}
	vm.syncActions()
	testutil.ErrorIf(t, !strings.Contains(vm.View(), want.Error()), "projection error not rendered: %s", vm.View())
	testutil.Equals(t, len(vm.actions), 0)
}
