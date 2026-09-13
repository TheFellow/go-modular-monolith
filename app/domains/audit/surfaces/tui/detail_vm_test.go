package tui_test

import (
	"strings"
	"testing"
	"time"

	auditmodels "github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	audittui "github.com/TheFellow/go-modular-monolith/app/domains/audit/surfaces/tui"
	ingredientsauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestDetailViewModel_ShowsEntryData(t *testing.T) {
	t.Parallel()
	start := time.Date(2024, 2, 1, 9, 30, 0, 0, time.UTC)
	end := start.Add(2 * time.Second)

	entry := auditmodels.AuditEntry{
		ID:          entity.NewAuditEntryID(),
		Action:      "ingredient.update",
		Resource:    entity.NewIngredientID().EntityUID(),
		Principal:   authn.Owner(),
		StartedAt:   start,
		CompletedAt: end,
		Success:     true,
	}

	detail := audittui.NewDetailViewModel(tuitest.DefaultListViewStyles[tui.ListViewStyles]())
	detail.SetSize(80, 40)
	detail.SetEntry(optional.Some(entry))

	view := detail.View()
	testutil.ErrorIf(t, !strings.Contains(view, entry.ID.String()), "expected entry id in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, entry.Action), "expected action in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, entry.Principal.String()), "expected principal in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, entry.Resource.String()), "expected resource in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, start.Format(time.RFC3339)), "expected start time in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, end.Format(time.RFC3339)), "expected completed time in view, got:\n%s", view)
	testutil.StringContains(t, view, "Duration: 2s")
	testutil.ErrorIf(t, !strings.Contains(view, "Success: true"), "expected success in view, got:\n%s", view)
}

func TestDetailViewModel_ShowsTouchedEntities(t *testing.T) {
	t.Parallel()
	touchA := entity.NewIngredientID().EntityUID()
	touchB := entity.NewDrinkID().EntityUID()

	entry := auditmodels.AuditEntry{
		ID:        entity.NewAuditEntryID(),
		Action:    "ingredient.update",
		Resource:  entity.NewIngredientID().EntityUID(),
		Principal: authn.Owner(),
		Touches:   []cedar.EntityUID{touchA, touchB},
	}

	detail := audittui.NewDetailViewModel(tuitest.DefaultListViewStyles[tui.ListViewStyles]())
	detail.SetEntry(optional.Some(entry))
	detail.SetSize(80, 40)

	view := detail.View()
	testutil.ErrorIf(t, !strings.Contains(view, "- "+touchA.String()), "expected touched entity in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "- "+touchB.String()), "expected touched entity in view, got:\n%s", view)
}

func TestDetailViewModel_NilEntry(t *testing.T) {
	t.Parallel()
	detail := audittui.NewDetailViewModel(tuitest.DefaultListViewStyles[tui.ListViewStyles]())
	detail.SetEntry(optional.None[auditmodels.AuditEntry]())

	view := detail.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Select an entry"), "expected placeholder view, got:\n%s", view)
}

func TestDetailViewModel_SetSize(t *testing.T) {
	t.Parallel()
	entry := auditmodels.AuditEntry{
		ID:        entity.NewAuditEntryID(),
		Action:    "ingredient.create",
		Resource:  entity.NewIngredientID().EntityUID(),
		Principal: authn.Owner(),
		Success:   true,
	}

	detail := audittui.NewDetailViewModel(tuitest.DefaultListViewStyles[tui.ListViewStyles]())
	detail.SetEntry(optional.Some(entry))
	detail.SetSize(20, 10)

	view := detail.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view after resizing")
}

func TestDetailViewModelExplainsAttemptedEffectsAndEmptySections(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ingredient := createIngredient(t, f)
	_, err := f.Ingredients.Retire(f.OwnerContext(), ingredient.ID, ingredientsmodels.Retirement{Reason: "discontinued"})
	testutil.Ok(t, err)
	entry := auditEntryFor(t, f, ingredientsauthz.ActionRetire, ingredient.ID.EntityUID())
	detail := audittui.NewDetailViewModel(tuitest.DefaultListViewStyles[tui.ListViewStyles]())
	detail.SetEntry(optional.Some(entry))
	view := detail.View()
	testutil.StringContains(t, view, "Effects\nCommitted effects")
	testutil.StringContains(t, view, "Before: ")
	testutil.StringContains(t, view, "After: ")
	// The same persisted explanation must never imply a commit after rollback.
	entry.Success = false
	detail.SetEntry(optional.Some(entry))
	testutil.StringContains(t, detail.View(), "Effects\nAttempted effects (not committed)")
	entry.Effects = nil
	entry.Participants = nil
	entry.WorkflowID = ""
	detail.SetEntry(optional.Some(entry))
	view = detail.View()
	testutil.StringContains(t, view, "Workflow: (none)")
	testutil.StringContains(t, view, "Referenced entities\n(none)")
	testutil.StringContains(t, view, "Effects\n(none)")
}
