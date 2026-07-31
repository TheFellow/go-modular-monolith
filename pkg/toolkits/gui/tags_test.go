//nolint:paralleltest // Fyne's application driver and focus state are process-global.
package gui

import (
	"errors"
	"testing"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	frameworktest "fyne.io/fyne/v2/test"
)

func TestTagPillsWrapAndRepresentEmptyTags(t *testing.T) {
	startTestApp(t)
	pills := TagPills([]string{"featured", "env=development", "region=west"})
	pills.Resize(framework.NewSize(150, 100))
	container := pills.(*framework.Container)
	if len(container.Objects) != 3 || container.Objects[2].Position().Y <= container.Objects[0].Position().Y {
		t.Fatalf("tag pills did not wrap: %#v", container.Objects)
	}
	if TagPills(nil).MinSize().IsZero() {
		t.Fatal("empty tags need a visible state")
	}
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
	if got := editor.CSV(); got != "featured,region=east" || editor.Input.Text != "" || changes != 1 {
		t.Fatalf("submitted editor = %q, pending %q, changes %d", got, editor.Input.Text, changes)
	}
	editor.Input.SetText("invalid")
	editor.Input.OnSubmitted(editor.Input.Text)
	if editor.ValidationError() == nil || editor.Input.Text != "invalid" || editor.CSV() != "featured,region=east" {
		t.Fatal("invalid token was not retained with its validation error")
	}
	pill := editor.Content.Objects[1].(*removableTagPill)
	if !pill.remove.Hidden {
		t.Fatal("remove control should be hidden until hover")
	}
	pill.MouseIn(&desktop.MouseEvent{})
	if pill.remove.Hidden {
		t.Fatal("remove control was not shown on hover")
	}
	frameworktest.Tap(pill.remove)
	if got := editor.CSV(); got != "featured" {
		t.Fatalf("removed editor = %q", got)
	}
}

func TestTagPillsCSVTrimsAndOmitsEmptyValues(t *testing.T) {
	startTestApp(t)
	pills := TagPillsCSV(" featured, region=west, ").(*framework.Container)
	if len(pills.Objects) != 2 {
		t.Fatalf("pill count = %d, want 2", len(pills.Objects))
	}
}

func TestTagPillColumnWidthPreservesLongAndMultiplePills(t *testing.T) {
	startTestApp(t)
	long := "environment=a-very-long-production-environment-name"
	want := compactTagPill(long).MinSize().Width + tagPillGap + compactTagPill("featured").MinSize().Width
	if got := TagPillColumnWidth([]string{long + ",featured"}, 100); got < want {
		t.Fatalf("tag column width %v clips natural pill row width %v", got, want)
	}
}

func TestTagPreviewRefreshesEditablePills(t *testing.T) {
	startTestApp(t)
	preview := NewTagPreview("classic")
	preview.SetCSV("featured, region=west")
	pills := preview.Content.Objects[0].(*framework.Container)
	if len(pills.Objects) != 2 {
		t.Fatalf("refreshed pill count = %d, want 2", len(pills.Objects))
	}
	if editor := TagEditor(preview, NewEntry("tags")); len(editor.(*framework.Container).Objects) != 2 {
		t.Fatal("tag editor must retain preview and input")
	}
}
