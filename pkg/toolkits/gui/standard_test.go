package gui_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	gui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

func TestStandardListPageComposesDeclaredRegions(t *testing.T) {
	t.Parallel()
	app := test.NewApp()
	t.Cleanup(app.Quit)

	page := gui.StandardListPage(gui.ListPage{
		Title: "Drinks", Subtitle: "Browse recipes", Filters: widget.NewLabel("FILTERS"),
		CollectionActions: []fyne.CanvasObject{gui.Primary(gui.NewButton("new", "New", nil))},
		DetailActions:     []fyne.CanvasObject{gui.Destructive(gui.NewButton("delete", "Delete", nil))},
		List:              widget.NewLabel("LIST"), Detail: widget.NewLabel("DETAIL"),
		Status: widget.NewLabel("STATUS"), Paging: container.NewHBox(widget.NewLabel("PAGING")),
	})
	window := app.NewWindow("standard")
	window.SetContent(page)
	for _, text := range []string{"Drinks", "Browse recipes", "FILTERS", "New", "Delete", "LIST", "DETAIL", "STATUS", "PAGING"} {
		testutil.ErrorIf(t, !containsText(page, text), "standard page does not contain %q", text)
	}
}

func TestEmptyCollectionIsIntentionalAndUsesExplicitIcon(t *testing.T) { //nolint:paralleltest // Fyne app and driver state is process-global.
	app := test.NewApp()
	t.Cleanup(app.Quit)
	empty := gui.EmptyCollection(gui.IconEmpty, "No ingredients found", "Adjust the filter.")
	testutil.ErrorIf(t, !containsText(empty, "No ingredients found") || !containsText(empty, "Adjust the filter."), "%v", "empty collection omitted its guidance")
}

func TestDetailFormGivesEveryFieldTheSameWidth(t *testing.T) { //nolint:paralleltest // Fyne app and driver state is process-global.
	app := test.NewApp()
	t.Cleanup(app.Quit)
	first := gui.DetailField("Short", widget.NewEntry())
	second := gui.DetailField("A much longer label", widget.NewEntry())
	form := gui.DetailForm(first, second)
	form.Resize(fyne.NewSize(640, 240))
	testutil.ErrorIf(t, first.Size().Width != second.Size().Width || first.Size().Width != 640, "field widths = %v and %v, want 640", first.Size().Width, second.Size().Width)
}

func containsText(object fyne.CanvasObject, want string) bool {
	switch typed := object.(type) {
	case *widget.Label:
		return typed.Text == want
	case *gui.SemanticButton:
		return typed.Text == want
	case *fyne.Container:
		for _, child := range typed.Objects {
			if containsText(child, want) {
				return true
			}
		}
	case *container.Split:
		return containsText(typed.Leading, want) || containsText(typed.Trailing, want)
	}
	return false
}
