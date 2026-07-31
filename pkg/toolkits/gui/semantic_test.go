//nolint:paralleltest // Fyne's focus and canvas state are process-global.
package gui_test

import (
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
	gui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

func TestSemanticDriverInteractsWithActualWidgets(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	entry := gui.NewEntry("drink-name")
	tapped := false
	button := gui.NewButton("save-drink", "Save", func() { tapped = true })
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
	left := gui.NewEntry("left")
	right := gui.NewEntry("right")
	split := gui.ListDetail(left, right, .35)
	if split.Leading != left || split.Trailing != right || split.Offset != .35 {
		t.Fatalf("unexpected split: %#v", split)
	}
}
