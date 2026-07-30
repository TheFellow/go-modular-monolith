package fyne

import (
	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// ListDetail builds the common resizable master/detail geometry. It assigns no
// selection, loading, empty-state, or navigation policy.
func ListDetail(list, detail framework.CanvasObject, listRatio float64) *container.Split {
	split := container.NewHSplit(list, detail)
	split.Offset = listRatio
	return split
}
