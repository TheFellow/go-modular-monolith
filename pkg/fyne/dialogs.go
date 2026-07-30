package fyne

import (
	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type TagMutationMode string

const (
	PreserveTags TagMutationMode = "Preserve"
	ReplaceTags  TagMutationMode = "Replace"
	ClearTags    TagMutationMode = "Clear"
)

// TaggedConfirmer is an optional richer dialog capability used by mutation
// confirmations. Test doubles can implement it without rendering overlays.
type TaggedConfirmer interface {
	ConfirmTagged(title, message, current string, respond func(bool, TagMutationMode, string))
}

// ConfirmTagged uses the richer capability when available. A basic Dialogs
// implementation safely degrades to preserving the existing complete set.
func ConfirmTagged(dialogs Dialogs, title, message, current string, respond func(bool, TagMutationMode, string)) {
	if tagged, ok := dialogs.(TaggedConfirmer); ok {
		tagged.ConfirmTagged(title, message, current, respond)
		return
	}
	dialogs.Confirm(title, message, func(ok bool) { respond(ok, PreserveTags, current) })
}

// Dialogs is injected into views so decisions and errors can be tested without
// scraping framework overlays.
type Dialogs interface {
	Confirm(title, message string, respond func(bool))
	ShowError(error)
	ShowWarning(title, message string)
}

type WindowDialogs struct{ Window framework.Window }

var _ Dialogs = WindowDialogs{}
var _ TaggedConfirmer = WindowDialogs{}

func (d WindowDialogs) Confirm(title, message string, respond func(bool)) {
	dialog.ShowConfirm(title, message, respond, d.Window)
}

func (d WindowDialogs) ConfirmTagged(title, message, current string, respond func(bool, TagMutationMode, string)) {
	choice := widget.NewRadioGroup([]string{string(PreserveTags), string(ReplaceTags), string(ClearTags)}, nil)
	choice.Horizontal = true
	choice.SetSelected(string(PreserveTags))
	values := widget.NewEntry()
	values.SetText(current)
	values.Disable()
	choice.OnChanged = func(value string) {
		if TagMutationMode(value) == ReplaceTags {
			values.Enable()
		} else {
			values.Disable()
		}
	}
	content := container.NewVBox(widget.NewLabel(message), choice, widget.NewLabel("Complete tag set (CSV)"), values)
	dlg := dialog.NewCustomConfirm(title, "Continue", "Cancel", content, func(ok bool) {
		respond(ok, TagMutationMode(choice.Selected), values.Text)
	}, d.Window)
	dlg.Show()
}

func (d WindowDialogs) ShowError(err error) { dialog.ShowError(err, d.Window) }

func (d WindowDialogs) ShowWarning(title, message string) {
	dialog.ShowInformation(title, message, d.Window)
}
