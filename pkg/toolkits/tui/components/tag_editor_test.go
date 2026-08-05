package components

import (
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func parseTestTags(value string) (string, error) {
	if value == "invalid" {
		return "", errors.New("invalid tags")
	}
	return value, nil
}

func TestTagEditorSavesThroughInjectedAdapter(t *testing.T) {
	t.Parallel()
	var saved string
	editor := NewTagEditor(func(_ int, tags string) (string, error) {
		saved = tags
		return tags, nil
	}, parseTestTags, 42, "target", "old")
	testutil.Ok(t, editor.field.SetValue("new=value"))
	msg := editor.save()()
	result, ok := msg.(TagsSavedMsg[int, string])
	testutil.ErrorIf(t, !ok, "expected TagsSavedMsg, got %T", msg)
	testutil.Equals(t, saved, "new=value")
	testutil.Equals(t, result.Target, 42)
}

func TestTagEditorRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	editor := NewTagEditor(func(_ int, tags string) (string, error) { return tags, nil }, parseTestTags, 42, "target", "")
	testutil.Ok(t, editor.field.SetValue("invalid"))
	_, cmd := editor.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	testutil.ErrorIf(t, cmd != nil, "invalid tags should not start a mutation")
	testutil.ErrorIf(t, editor.err == nil, "expected validation error")
}
