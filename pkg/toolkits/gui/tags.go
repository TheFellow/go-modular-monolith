package gui

import (
	"image/color"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const tagPillGap float32 = 6

// TagPills renders canonical tag strings as compact, wrapping visual tokens.
func TagPills(values []string) framework.CanvasObject {
	if len(values) == 0 {
		return widget.NewLabel("None")
	}
	objects := make([]framework.CanvasObject, 0, len(values))
	for _, value := range values {
		label := widget.NewLabel(value)
		background := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
		background.CornerRadius = 14
		background.StrokeWidth = 1
		background.StrokeColor = withAlpha(theme.Color(theme.ColorNamePrimary), 180)
		objects = append(objects, container.NewStack(background, container.NewPadded(label)))
	}
	return container.New(&pillWrapLayout{}, objects...)
}

func withAlpha(value color.Color, alpha uint8) color.Color {
	r, g, b, _ := value.RGBA()
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: alpha}
}

type pillWrapLayout struct{}

func (*pillWrapLayout) MinSize(objects []framework.CanvasObject) framework.Size {
	var width, height float32
	for _, object := range objects {
		size := object.MinSize()
		width = max(width, size.Width)
		height = max(height, size.Height)
	}
	return framework.NewSize(width, height)
}

func (*pillWrapLayout) Layout(objects []framework.CanvasObject, size framework.Size) {
	var x, y, rowHeight float32
	for _, object := range objects {
		item := object.MinSize()
		if x > 0 && x+item.Width > size.Width {
			x, y, rowHeight = 0, y+rowHeight+tagPillGap, 0
		}
		object.Move(framework.NewPos(x, y))
		object.Resize(item)
		x += item.Width + tagPillGap
		rowHeight = max(rowHeight, item.Height)
	}
}
