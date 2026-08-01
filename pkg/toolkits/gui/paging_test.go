package gui_test

import (
	"testing"
	"time"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	gui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

func TestAutoPagingRowTableLoadsOnceWhenFinalRowIsRendered(t *testing.T) {
	rows := 25
	loaded := make(chan struct{}, 2)
	table := gui.NewAutoPagingRowTable(func() (int, int) { return rows, 2 }, func() framework.CanvasObject { return widget.NewLabel("") }, func(widget.TableCellID, framework.CanvasObject) {}, func() { loaded <- struct{}{} })
	cell := widget.NewLabel("")
	table.UpdateCell(widget.TableCellID{Row: 24, Col: 1}, cell)
	table.UpdateCell(widget.TableCellID{Row: 24, Col: 1}, cell)
	select {
	case <-loaded:
	case <-time.After(time.Second):
		t.Fatal("final row did not request another page")
	}
	select {
	case <-loaded:
		t.Fatal("same row count requested more than once")
	case <-time.After(10 * time.Millisecond):
	}
	rows = 50
	table.UpdateCell(widget.TableCellID{Row: 49, Col: 1}, cell)
	select {
	case <-loaded:
	case <-time.After(time.Second):
		t.Fatal("new final row did not request another page")
	}
	testutil.Equals(t, gui.PageLimit, 25)
}
