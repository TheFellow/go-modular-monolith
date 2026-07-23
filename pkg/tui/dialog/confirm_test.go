package dialog_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/dialog"
	tea "github.com/charmbracelet/bubbletea"
)

func TestConfirmDialogNavigation(t *testing.T) {
	t.Parallel()

	dlg := dialog.NewConfirmDialog("Title", "Message")
	dlg, _ = dlg.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated, cmd := dlg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	testutil.IsTrue(t, updated.IsCancelled())
	testutil.NotNil(t, cmd)
}

func TestConfirmDialogConfirm(t *testing.T) {
	t.Parallel()

	dlg := dialog.NewConfirmDialog("Title", "Message")
	updated, cmd := dlg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	testutil.IsTrue(t, updated.IsConfirmed())
	testutil.IsFalse(t, updated.IsCancelled())
	testutil.NotNil(t, cmd)
	testutil.Cast[dialog.ConfirmMsg](t, cmd())
}

func TestConfirmDialogCancel(t *testing.T) {
	t.Parallel()

	dlg := dialog.NewConfirmDialog("Title", "Message", dialog.WithFocusCancel())
	updated, cmd := dlg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	testutil.IsTrue(t, updated.IsCancelled())
	testutil.IsFalse(t, updated.IsConfirmed())
	testutil.NotNil(t, cmd)
	testutil.Cast[dialog.CancelMsg](t, cmd())
}

func TestConfirmDialogEscCancel(t *testing.T) {
	t.Parallel()

	dlg := dialog.NewConfirmDialog("Title", "Message")
	updated, cmd := dlg.Update(tea.KeyMsg{Type: tea.KeyEsc})
	testutil.IsTrue(t, updated.IsCancelled())
	testutil.NotNil(t, cmd)
	testutil.Cast[dialog.CancelMsg](t, cmd())
}
