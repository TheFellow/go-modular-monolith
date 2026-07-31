package gui_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	gui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

func TestStandardListPageComposesDeclaredRegions(t *testing.T) {
	t.Parallel()
	app := test.NewApp()
	t.Cleanup(app.Quit)

	page := gui.StandardListPage(gui.ListPage{
		Title: "Drinks", Subtitle: "Browse recipes", Filters: widget.NewLabel("FILTERS"),
		PrimaryActions: []fyne.CanvasObject{gui.Primary(gui.NewButton("new", "New", nil))},
		OtherActions:   []fyne.CanvasObject{gui.Destructive(gui.NewButton("delete", "Delete", nil))},
		List:           widget.NewLabel("LIST"), Detail: widget.NewLabel("DETAIL"),
		Status: widget.NewLabel("STATUS"), Paging: container.NewHBox(widget.NewLabel("PAGING")),
	})
	window := app.NewWindow("standard")
	window.SetContent(page)
	for _, text := range []string{"Drinks", "Browse recipes", "FILTERS", "New", "Delete", "LIST", "DETAIL", "STATUS", "PAGING"} {
		if !containsText(page, text) {
			t.Fatalf("standard page does not contain %q", text)
		}
	}
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
