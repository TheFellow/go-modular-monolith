package gui

import (
	"strings"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
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
	button.Icon = ActionIcon(id, label)
	button.OnTapped = tapped
	button.ExtendBaseWidget(button)
	return button
}

// ButtonWithIcon overrides the shared action convention for the occasional
// domain action whose meaning cannot be inferred from its stable ID or label.
// Text is deliberately retained: icons reinforce actions; they never replace
// their accessible, translatable label.
func ButtonWithIcon(button *SemanticButton, icon framework.Resource) *SemanticButton {
	button.Icon = icon
	button.Refresh()
	return button
}

// ActionIcon is the application-wide symbol vocabulary for repeated actions.
// Semantic IDs are considered before copy so tests and future wording changes
// do not alter meaning.
func ActionIcon(id, label string) framework.Resource {
	meaning := strings.ToLower(id + " " + label)
	switch {
	case containsAny(meaning, "tag"):
		return theme.MailAttachmentIcon()
	case containsAny(meaning, "delete", "remove"):
		return theme.DeleteIcon()
	case containsAny(meaning, "cancel", "close", "back"):
		return theme.NavigateBackIcon()
	case containsAny(meaning, "save", "submit", "place order", "orders.place"):
		return theme.DocumentSaveIcon()
	case containsAny(meaning, "refresh", "reload"):
		return theme.ViewRefreshIcon()
	case containsAny(meaning, "previous", " prev"):
		return theme.NavigateBackIcon()
	case containsAny(meaning, "next"):
		return theme.NavigateNextIcon()
	case containsAny(meaning, "search", "apply", "filter"):
		return theme.SearchIcon()
	case containsAny(meaning, "new", "create", "add", "adjust"):
		return theme.ContentAddIcon()
	case containsAny(meaning, "edit", "rename", "set"):
		return theme.DocumentCreateIcon()
	case containsAny(meaning, "publish", "complete", "confirm"):
		return theme.ConfirmIcon()
	case containsAny(meaning, "analy", "open", "select", "view"):
		return theme.VisibilityIcon()
	default:
		return nil
	}
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
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
