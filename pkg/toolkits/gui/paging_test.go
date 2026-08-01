package gui_test

import (
	"testing"
	"testing/synctest"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	gui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

func TestAutoPagingRowTableLoadsOnceWhenFinalRowIsRendered(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rows, loads := 25, 0
		table := gui.NewAutoPagingRowTable(func() (int, int) { return rows, 2 }, func() framework.CanvasObject { return widget.NewLabel("") }, func(widget.TableCellID, framework.CanvasObject) {}, func() { loads++ })
		cell := widget.NewLabel("")
		table.UpdateCell(widget.TableCellID{Row: 24, Col: 1}, cell)
		table.UpdateCell(widget.TableCellID{Row: 24, Col: 1}, cell)
		synctest.Wait()
		testutil.Equals(t, loads, 1)

		rows = 50
		table.UpdateCell(widget.TableCellID{Row: 49, Col: 1}, cell)
		synctest.Wait()
		testutil.Equals(t, loads, 2)
		testutil.Equals(t, gui.PageLimit, 25)
	})
}
