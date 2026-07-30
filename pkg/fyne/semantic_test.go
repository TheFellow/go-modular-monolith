//nolint:paralleltest // Fyne's focus and canvas state are process-global.
package fyne_test

import (
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"

	fyneui "github.com/TheFellow/go-modular-monolith/pkg/fyne"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
)

func TestSemanticDriverInteractsWithActualWidgets(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	entry := fyneui.NewEntry("drink-name")
	tapped := false
	button := fyneui.NewButton("save-drink", "Save", func() { tapped = true })
	driver := fynetest.NewDriver(t, container.NewVBox(entry, button))

	driver.Type("drink-name", "Gimlet")
	driver.Tap("save-drink")
	if entry.Text != "Gimlet" || !tapped {
		t.Fatalf("entry=%q tapped=%v", entry.Text, tapped)
	}
}

func TestListDetailPreservesSuppliedObjectsAndRatio(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	left := fyneui.NewEntry("left")
	right := fyneui.NewEntry("right")
	split := fyneui.ListDetail(left, right, .35)
	if split.Leading != left || split.Trailing != right || split.Offset != .35 {
		t.Fatalf("unexpected split: %#v", split)
	}
}
