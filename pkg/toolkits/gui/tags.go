package gui

import (
	"encoding/csv"
	"image/color"
	"io"
	"sort"
	"strings"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	tagPillGap       float32 = 6
	tagInputMinWidth float32 = 220
)

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

// TagTokenEditor edits a tag collection as tokens instead of exposing its CSV
// transport representation. Enter commits the pending key or key=value token;
// a token with an existing key replaces that token.
type TagTokenEditor struct {
	Content   *framework.Container
	Input     *SemanticEntry
	OnChanged func(string)
	// Normalize validates and upserts input into the current serialized set.
	// Applications should provide their strongly typed tag-domain operation.
	Normalize func(current, input string) (string, error)

	id     string
	values []string
	err    error
}

// NewTagTokenEditor creates an editor with semantic IDs for its input and
// remove controls. A remove control has the ID "<id>.remove.<key>".
func NewTagTokenEditor(id, value string) *TagTokenEditor {
	e := &TagTokenEditor{id: id, Input: NewEntry(id)}
	e.Input.SetPlaceHolder("Add tag and press Enter")
	e.Input.OnSubmitted = func(value string) { e.add(value) }
	e.SetCSV(value)
	return e
}

// CSV returns the canonical serialized value expected by existing presenters.
func (e *TagTokenEditor) CSV() string { return formatTagCSV(e.values) }

// ValidationError reports the most recent rejected pending token.
func (e *TagTokenEditor) ValidationError() error { return e.err }

// SetCSV replaces the token set without reporting a user edit.
func (e *TagTokenEditor) SetCSV(value string) {
	e.values = parseTagCSV(value)
	e.rebuild()
}

// SetEnabled controls the input and token removal controls.
func (e *TagTokenEditor) SetEnabled(enabled bool) {
	if enabled {
		e.Input.Enable()
	} else {
		e.Input.Disable()
	}
	for _, object := range e.Content.Objects {
		if pill, ok := object.(*removableTagPill); ok {
			pill.setEnabled(enabled)
		}
	}
}

func (e *TagTokenEditor) add(value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if e.Normalize != nil {
		next, err := e.Normalize(e.CSV(), value)
		if err != nil {
			e.err = err
			e.Input.SetValidationError(err)
			return
		}
		e.err = nil
		e.Input.SetValidationError(nil)
		e.SetCSV(next)
		e.Input.SetText("")
		if e.OnChanged != nil {
			e.OnChanged(e.CSV())
		}
		return
	}
	key := tagKey(value)
	if key == "" {
		return
	}
	replaced := false
	for index, existing := range e.values {
		if tagKey(existing) == key {
			e.values[index], replaced = value, true
			break
		}
	}
	if !replaced {
		e.values = append(e.values, value)
	}
	sort.SliceStable(e.values, func(i, j int) bool { return tagKey(e.values[i]) < tagKey(e.values[j]) })
	e.Input.SetText("")
	e.changed()
}

func (e *TagTokenEditor) remove(key string) {
	for index, value := range e.values {
		if tagKey(value) == key {
			e.values = append(e.values[:index], e.values[index+1:]...)
			e.changed()
			return
		}
	}
}

func (e *TagTokenEditor) changed() {
	e.rebuild()
	if e.OnChanged != nil {
		e.OnChanged(e.CSV())
	}
}

func (e *TagTokenEditor) rebuild() {
	objects := make([]framework.CanvasObject, 0, len(e.values)+1)
	for _, value := range e.values {
		key := tagKey(value)
		objects = append(objects, newRemovableTagPill(e.id+".remove."+key, value, func() { e.remove(key) }))
	}
	objects = append(objects, e.Input)
	if e.Content == nil {
		e.Content = container.New(&pillWrapLayout{}, objects...)
	} else {
		e.Content.Objects = objects
		e.Content.Refresh()
	}
}

func tagKey(value string) string {
	key, _, _ := strings.Cut(value, "=")
	return strings.TrimSpace(key)
}

func parseTagCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	r := csv.NewReader(strings.NewReader(value))
	r.TrimLeadingSpace = true
	values, err := r.Read()
	if err != nil {
		return []string{strings.TrimSpace(value)}
	}
	if _, err = r.Read(); err != io.EOF {
		return []string{strings.TrimSpace(value)}
	}
	result := values[:0]
	for _, part := range values {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

func formatTagCSV(values []string) string {
	if len(values) == 0 {
		return ""
	}
	var output strings.Builder
	w := csv.NewWriter(&output)
	_ = w.Write(values)
	w.Flush()
	return strings.TrimSuffix(output.String(), "\n")
}

type removableTagPill struct {
	widget.BaseWidget
	label   string
	remove  *SemanticButton
	enabled bool
}

func (p *removableTagPill) SemanticChildren() []framework.CanvasObject {
	return []framework.CanvasObject{p.remove}
}

func newRemovableTagPill(id, label string, remove func()) *removableTagPill {
	p := &removableTagPill{label: label, remove: NewButton(id, "×", remove), enabled: true}
	p.remove.Importance = widget.LowImportance
	p.remove.Hide()
	p.ExtendBaseWidget(p)
	return p
}

func (p *removableTagPill) MouseIn(*desktop.MouseEvent) {
	if p.enabled {
		p.remove.Show()
		p.Refresh()
	}
}

func (p *removableTagPill) MouseMoved(*desktop.MouseEvent) {}

func (p *removableTagPill) MouseOut() {
	p.remove.Hide()
	p.Refresh()
}

func (p *removableTagPill) setEnabled(enabled bool) {
	p.enabled = enabled
	if !enabled {
		p.remove.Hide()
	}
}

func (p *removableTagPill) CreateRenderer() framework.WidgetRenderer {
	label := widget.NewLabel(p.label)
	background := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	background.CornerRadius = 14
	background.StrokeWidth = 1
	background.StrokeColor = withAlpha(theme.Color(theme.ColorNamePrimary), 180)
	return &tagPillRenderer{pill: p, background: background, label: label, objects: []framework.CanvasObject{background, label, p.remove}}
}

type tagPillRenderer struct {
	pill       *removableTagPill
	background *canvas.Rectangle
	label      *widget.Label
	objects    []framework.CanvasObject
}

func (r *tagPillRenderer) Layout(size framework.Size) {
	r.background.Resize(size)
	r.label.Move(framework.NewPos(theme.Padding(), 0))
	r.label.Resize(framework.NewSize(size.Width-theme.Padding()*2-16, size.Height))
	buttonSize := framework.NewSize(22, 22)
	r.pill.remove.Move(framework.NewPos(size.Width-buttonSize.Width, 0))
	r.pill.remove.Resize(buttonSize)
}
func (r *tagPillRenderer) MinSize() framework.Size {
	s := r.label.MinSize()
	return framework.NewSize(s.Width+theme.Padding()*2+16, max(s.Height+theme.Padding(), float32(22)))
}
func (r *tagPillRenderer) Refresh()                          { r.background.Refresh(); r.label.Refresh() }
func (r *tagPillRenderer) Objects() []framework.CanvasObject { return r.objects }
func (r *tagPillRenderer) Destroy()                          {}

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
	return TagPills(parseTagCSV(value))
}

func compactTagPill(value string) framework.CanvasObject {
	label := widget.NewLabel(value)
	label.Truncation = framework.TextTruncateEllipsis
	background := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	background.CornerRadius = 10
	background.StrokeWidth = 1
	background.StrokeColor = withAlpha(theme.Color(theme.ColorNamePrimary), 150)
	return container.New(&compactPillLayout{}, background, label)
}

// TagPillColumnWidth returns the width required to display the widest tag row
// without shortening an individual pill. The table can then scroll
// horizontally when that natural width is wider than the viewport.
func TagPillColumnWidth(values []string, minimum float32) float32 {
	width := minimum
	for _, value := range values {
		rowWidth := float32(0)
		for i, tag := range parseTagCSV(value) {
			if i > 0 {
				rowWidth += tagPillGap
			}
			rowWidth += compactTagPill(tag).MinSize().Width
		}
		width = max(width, rowWidth)
	}
	return width
}

type compactPillLayout struct{}

func (*compactPillLayout) MinSize(objects []framework.CanvasObject) framework.Size {
	label := objects[1].(*widget.Label)
	measured := framework.MeasureText(label.Text, theme.TextSize(), label.TextStyle)
	// Fyne's Label renderer reserves more horizontal inset than MeasureText
	// reports. Leave enough room for short tags to render in full while still
	// bounding long values to their table column.
	return framework.NewSize(measured.Width+40, measured.Height+2)
}

func (*compactPillLayout) Layout(objects []framework.CanvasObject, size framework.Size) {
	objects[0].Move(framework.NewPos(0, 2))
	objects[0].Resize(framework.NewSize(size.Width, max(0, size.Height-4)))
	objects[1].Move(framework.NewPos(6, 0))
	objects[1].Resize(framework.NewSize(max(0, size.Width-12), size.Height))
}

// compactPillRowLayout preserves each pill's natural width. Table tag columns
// start wide enough for their content and can be widened with the native header
// divider instead of silently clipping a tag's text.
type compactPillRowLayout struct{}

func (*compactPillRowLayout) MinSize([]framework.CanvasObject) framework.Size {
	return framework.NewSize(0, theme.Size(theme.SizeNameInlineIcon))
}

func (*compactPillRowLayout) Layout(objects []framework.CanvasObject, size framework.Size) {
	x := float32(0)
	for i, object := range objects {
		gap := float32(0)
		if i > 0 {
			gap = tagPillGap
		}
		width := object.MinSize().Width
		object.Move(framework.NewPos(x+gap, 0))
		object.Resize(framework.NewSize(width, size.Height))
		x += gap + width
	}
}

func withAlpha(value color.Color, alpha uint8) color.Color {
	r, g, b, _ := value.RGBA()
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: alpha}
}

type pillWrapLayout struct{}

func (*pillWrapLayout) MinSize(objects []framework.CanvasObject) framework.Size {
	var width, height float32
	for _, object := range objects {
		size := tagTokenSize(object)
		width = max(width, size.Width)
		height = max(height, size.Height)
	}
	return framework.NewSize(width, height)
}

func (*pillWrapLayout) Layout(objects []framework.CanvasObject, size framework.Size) {
	var x, y, rowHeight float32
	for _, object := range objects {
		item := tagTokenSize(object)
		if x > 0 && x+item.Width > size.Width {
			x, y, rowHeight = 0, y+rowHeight+tagPillGap, 0
		}
		object.Move(framework.NewPos(x, y))
		object.Resize(item)
		x += item.Width + tagPillGap
		rowHeight = max(rowHeight, item.Height)
	}
}

func tagTokenSize(object framework.CanvasObject) framework.Size {
	size := object.MinSize()
	if _, ok := object.(*SemanticEntry); ok {
		size.Width = max(size.Width, tagInputMinWidth)
	}
	return size
}
