package tui_test

import (
	"strings"
	"testing"
	"time"

	auditdao "github.com/TheFellow/go-modular-monolith/app/domains/audit/internal/dao"
	auditmodels "github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	audittui "github.com/TheFellow/go-modular-monolith/app/domains/audit/surfaces/tui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mjl-/bstore"
)

func TestListViewModel_ShowsEntriesAfterLoad(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	uid := entity.NewIngredientID().EntityUID()
	entry := &auditmodels.AuditEntry{
		ID:          entity.NewAuditEntryID(),
		Action:      "ingredient.create",
		Resource:    uid,
		Principal:   authn.Owner(),
		StartedAt:   time.Now().UTC(),
		CompletedAt: time.Now().UTC(),
		Success:     true,
	}
	insertAuditEntry(t, f, *entry)

	model := tuitest.InitAndLoad(t, audittui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, "ingredient.create"), "expected view to contain entry action, got:\n%s", view)
}

func TestListViewModel_ShowsLoadingState(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	model := audittui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	)
	_ = model.Init()

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Loading"), "expected loading state, got:\n%s", view)
}

func TestListViewModel_ShowsEmptyState(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	model := tuitest.InitAndLoad(t, audittui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view for empty list")
}

func TestListViewModel_ShowsTimestampAndAction(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	start := time.Date(2024, 2, 1, 9, 30, 0, 0, time.UTC)
	uid := entity.NewIngredientID().EntityUID()
	entry := &auditmodels.AuditEntry{
		ID:          entity.NewAuditEntryID(),
		Action:      "ingredient.update",
		Resource:    uid,
		Principal:   authn.Owner(),
		StartedAt:   start,
		CompletedAt: start.Add(2 * time.Second),
		Success:     true,
	}
	insertAuditEntry(t, f, *entry)

	model := tuitest.InitAndLoad(t, audittui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, "ingredient.update"), "expected action in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "09:30:00"), "expected timestamp in view, got:\n%s", view)
}

func TestListViewModel_SetSize_NarrowWidth(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	uid := entity.NewIngredientID().EntityUID()
	entry := &auditmodels.AuditEntry{
		ID:          entity.NewAuditEntryID(),
		Action:      "ingredient.delete",
		Resource:    uid,
		Principal:   authn.Owner(),
		StartedAt:   time.Now().UTC(),
		CompletedAt: time.Now().UTC(),
		Success:     true,
	}
	insertAuditEntry(t, f, *entry)

	model := tuitest.InitAndLoad(t, audittui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 30, Height: 20})

	view := model.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view for narrow width")
}

func insertAuditEntry(t testing.TB, f *testutil.Fixture, entry auditmodels.AuditEntry) {
	t.Helper()
	err := f.Store.Write(f.OwnerContext(), func(tx *bstore.Tx) error {
		ctx := middleware.NewContext(f.OwnerContext(), middleware.WithTransaction(tx))
		return auditdao.New().Insert(ctx, entry)
	})
	testutil.Ok(t, err)
}
