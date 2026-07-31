package gui

import (
	"image/color"
	"strings"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const tagPillGap float32 = 6

// TagPreview is a replaceable pill collection for an editable tag field.
type TagPreview struct {
	Content *framework.Container
}

func NewTagPreview(value string) *TagPreview {
	preview := &TagPreview{Content: container.NewStack()}
	preview.SetCSV(value)
	return preview
}

// SetCSV refreshes the pills without replacing the surrounding form field.
func (p *TagPreview) SetCSV(value string) {
	p.Content.RemoveAll()
	p.Content.Add(TagPillsCSV(value))
	p.Content.Refresh()
}

// TagEditor presents the visual tokens and retains the editable canonical
// representation directly beneath them.
func TagEditor(preview *TagPreview, entry framework.CanvasObject) framework.CanvasObject {
	return container.NewVBox(preview.Content, entry)
}

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

// TagPillsCSV renders the canonical comma-separated representation used by
// domain forms without making the GUI toolkit depend on the tag domain.
func TagPillsCSV(value string) framework.CanvasObject {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			values = append(values, part)
		}
	}
	return TagPills(values)
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
