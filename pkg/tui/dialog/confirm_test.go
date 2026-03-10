package dialog_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/tui/dialog"
	tea "github.com/charmbracelet/bubbletea"
)

func TestConfirmDialogNavigation(t *testing.T) {
	dlg := dialog.NewConfirmDialog("Title", "Message")
	dlg, _ = dlg.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated, cmd := dlg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !updated.IsCancelled() {
		t.Fatalf("expected cancel after switching focus and pressing enter")
	}
	if cmd == nil {
		t.Fatalf("expected cancel cmd after switching focus")
	}
}

func TestConfirmDialogConfirm(t *testing.T) {
	dlg := dialog.NewConfirmDialog("Title", "Message")
	updated, cmd := dlg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !updated.IsConfirmed() {
		t.Fatalf("expected confirmed after enter")
	}
	if updated.IsCancelled() {
		t.Fatalf("did not expect cancelled after enter")
	}
	if cmd == nil {
		t.Fatalf("expected confirm cmd")
	}
	msg := cmd()
	if _, ok := msg.(dialog.ConfirmMsg); !ok {
		t.Fatalf("expected ConfirmMsg")
	}
}

func TestConfirmDialogCancel(t *testing.T) {
	dlg := dialog.NewConfirmDialog("Title", "Message", dialog.WithFocusCancel())
	updated, cmd := dlg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !updated.IsCancelled() {
		t.Fatalf("expected cancelled after enter when focus cancel")
	}
	if updated.IsConfirmed() {
		t.Fatalf("did not expect confirmed")
	}
	if cmd == nil {
		t.Fatalf("expected cancel cmd")
	}
	msg := cmd()
	if _, ok := msg.(dialog.CancelMsg); !ok {
		t.Fatalf("expected CancelMsg")
	}
}

func TestConfirmDialogEscCancel(t *testing.T) {
	dlg := dialog.NewConfirmDialog("Title", "Message")
	updated, cmd := dlg.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !updated.IsCancelled() {
		t.Fatalf("expected cancelled after esc")
	}
	if cmd == nil {
		t.Fatalf("expected cancel cmd")
	}
	msg := cmd()
	if _, ok := msg.(dialog.CancelMsg); !ok {
		t.Fatalf("expected CancelMsg")
	}
}
