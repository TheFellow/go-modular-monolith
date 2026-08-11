package gui

import (
	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// Semantic identifies an interactive control independently of its displayed
// copy, allowing tests to drive real widgets without depending on layout.
type Semantic interface{ SemanticID() string }

type SemanticButton struct {
	widget.Button
	id string
}

func NewButton(id, label string, tapped func()) *SemanticButton {
	button := &SemanticButton{id: id}
	button.Text = label
	button.OnTapped = tapped
	button.ExtendBaseWidget(button)
	return button
}

func (b *SemanticButton) SemanticID() string { return b.id }

type SemanticEntry struct {
	widget.Entry
	id string
}

func NewEntry(id string) *SemanticEntry {
	entry := &SemanticEntry{id: id}
	// A single-line field must not install an inner scroll target; otherwise it
	// consumes wheel events intended for the surrounding form page.
	entry.Scroll = framework.ScrollNone
	entry.ExtendBaseWidget(entry)
	return entry
}

// NewMultiLineEntry keeps a vertical inner scroll target so long text remains
// independently scrollable inside a scrolling form page.
func NewMultiLineEntry(id string) *SemanticEntry {
	entry := NewEntry(id)
	entry.MultiLine = true
	entry.Scroll = framework.ScrollVerticalOnly
	return entry
}

func (e *SemanticEntry) SemanticID() string { return e.id }
