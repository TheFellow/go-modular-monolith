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
