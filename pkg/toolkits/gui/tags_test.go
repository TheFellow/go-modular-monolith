//nolint:paralleltest // Fyne's application driver and focus state are process-global.
package gui

import (
	"errors"
	"testing"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	frameworktest "fyne.io/fyne/v2/test"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestTagPillsWrapAndRepresentEmptyTags(t *testing.T) {
	startTestApp(t)
	pills := TagPills([]string{"featured", "env=development", "region=west"})
	pills.Resize(framework.NewSize(150, 100))
	container := pills.(*framework.Container)
	testutil.ErrorIf(t, len(container.Objects) != 3 || container.Objects[2].Position().Y <= container.Objects[0].Position().Y, "tag pills did not wrap: %#v", container.Objects)
	testutil.ErrorIf(t, TagPills(nil).MinSize().IsZero(), "%v", "empty tags need a visible state")
}

func TestTagTokenEditorSubmitsReplacesRejectsAndRemovesTokens(t *testing.T) {
	startTestApp(t)
	editor := NewTagTokenEditor("tags", "featured,region=west")
	editor.Normalize = func(current, input string) (string, error) {
		if input == "invalid" {
			return "", errors.New("invalid tag")
		}
		if input == "region=east" {
			return "featured,region=east", nil
		}
		return current + "," + input, nil
	}
	changes := 0
	editor.OnChanged = func(string) { changes++ }
	editor.Input.SetText("region=east")
	editor.Input.OnSubmitted(editor.Input.Text)
	{
		got := editor.CSV()
		testutil.ErrorIf(t, got != "featured,region=east" || editor.Input.Text != "" || changes != 1, "submitted editor = %q, pending %q, changes %d", got, editor.Input.Text, changes)
	}
	editor.Input.SetText("invalid")
	editor.Input.OnSubmitted(editor.Input.Text)
	testutil.ErrorIf(t, editor.ValidationError() == nil || editor.Input.Text != "invalid" || editor.CSV() != "featured,region=east", "%v", "invalid token was not retained with its validation error")
	pill := editor.Content.Objects[1].(*removableTagPill)
	testutil.ErrorIf(t, !pill.remove.Hidden, "%v", "remove control should be hidden until hover")
	pill.MouseIn(&desktop.MouseEvent{})
	testutil.ErrorIf(t, pill.remove.Hidden, "%v", "remove control was not shown on hover")
	frameworktest.Tap(pill.remove)
	{
		got := editor.CSV()
		testutil.ErrorIf(t, got != "featured", "removed editor = %q", got)
	}
}

func TestTagPillsCSVTrimsAndOmitsEmptyValues(t *testing.T) {
	startTestApp(t)
	pills := TagPillsCSV(" featured, region=west, ").(*framework.Container)
	testutil.ErrorIf(t, len(pills.Objects) != 2, "pill count = %d, want 2", len(pills.Objects))
}

func TestTagPillColumnWidthPreservesLongAndMultiplePills(t *testing.T) {
	startTestApp(t)
	long := "environment=a-very-long-production-environment-name"
	want := compactTagPill(long).MinSize().Width + tagPillGap + compactTagPill("featured").MinSize().Width
	{
		got := TagPillColumnWidth([]string{long + ",featured"}, 100)
		testutil.ErrorIf(t, got < want, "tag column width %v clips natural pill row width %v", got, want)
	}
}

func TestTagPreviewRefreshesEditablePills(t *testing.T) {
	startTestApp(t)
	preview := NewTagPreview("classic")
	preview.SetCSV("featured, region=west")
	pills := preview.Content.Objects[0].(*framework.Container)
	testutil.ErrorIf(t, len(pills.Objects) != 2, "refreshed pill count = %d, want 2", len(pills.Objects))
	{
		editor := TagEditor(preview, NewEntry("tags"))
		testutil.ErrorIf(t, len(editor.(*framework.Container).Objects) != 2, "%v", "tag editor must retain preview and input")
	}
}

func TestDisabledTagTokenEditorRejectsSubmitAndRemovalAcrossRebuild(t *testing.T) {
	startTestApp(t)
	editor := NewTagTokenEditor("tags", "featured")
	changes := 0
	editor.OnChanged = func(string) { changes++ }
	editor.SetEnabled(false)

	editor.Input.OnSubmitted("new")
	pill := editor.Content.Objects[0].(*removableTagPill)
	pill.remove.OnTapped()
	editor.SetCSV("featured,region=west")
	for _, object := range editor.Content.Objects[:2] {
		{
			remove := object.(*removableTagPill).remove
			testutil.ErrorIf(t, !remove.Disabled(), "%v", "SetCSV recreated an enabled remove control in a read-only editor")
		}
	}
	{
		got := editor.CSV()
		testutil.ErrorIf(t, got != "featured,region=west" || changes != 0, "disabled editor = %q changes=%d", got, changes)
	}
}

func TestTagTokenRemovalInvokesChangeOnceAndCannotRepeat(t *testing.T) {
	startTestApp(t)
	editor := NewTagTokenEditor("tags", "featured")
	changes := 0
	editor.OnChanged = func(string) { changes++ }
	pill := editor.Content.Objects[0].(*removableTagPill)
	pill.remove.OnTapped()
	pill.remove.OnTapped() // stale UI callback after the pill has been rebuilt.
	{
		got := editor.CSV()
		testutil.ErrorIf(t, got != "" || changes != 1, "editor=%q changes=%d", got, changes)
	}
}
