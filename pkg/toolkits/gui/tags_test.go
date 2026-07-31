//nolint:paralleltest // Fyne's application driver and focus state are process-global.
package gui

import (
	"testing"

	framework "fyne.io/fyne/v2"
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

func TestTagPillsCSVTrimsAndOmitsEmptyValues(t *testing.T) {
	startTestApp(t)
	pills := TagPillsCSV(" featured, region=west, ").(*framework.Container)
	if len(pills.Objects) != 2 {
		t.Fatalf("pill count = %d, want 2", len(pills.Objects))
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
