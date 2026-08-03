package gui

import "fyne.io/fyne/v2/widget"

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
	entry.ExtendBaseWidget(entry)
	return entry
}

func (e *SemanticEntry) SemanticID() string { return e.id }
