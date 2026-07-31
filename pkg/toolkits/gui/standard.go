package gui

import (
	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// ListPage describes the standard primary/secondary workspace used by
// catalog-style application surfaces. Domains provide content and commands;
// the toolkit owns visual hierarchy and placement.
type ListPage struct {
	Title, Subtitle string
	Filters         framework.CanvasObject
	PrimaryActions  []framework.CanvasObject
	OtherActions    []framework.CanvasObject
	List, Detail    framework.CanvasObject
	Status, Paging  framework.CanvasObject
	ListRatio       float64
}

// StandardListPage builds a consistent page header, action bar, filter area,
// master/detail body, and status footer.
func StandardListPage(page ListPage) framework.CanvasObject {
	title := widget.NewLabelWithStyle(page.Title, framework.TextAlignLeading, framework.TextStyle{Bold: true})
	heading := []framework.CanvasObject{title}
	if page.Subtitle != "" {
		subtitle := widget.NewLabel(page.Subtitle)
		subtitle.Wrapping = framework.TextWrapWord
		heading = append(heading, subtitle)
	}
	if actions := ActionBar(page.PrimaryActions, page.OtherActions); actions != nil {
		heading = append(heading, actions)
	}
	if page.Filters != nil {
		heading = append(heading, widget.NewSeparator(), page.Filters)
	}
	ratio := page.ListRatio
	if ratio <= 0 || ratio >= 1 {
		ratio = .38
	}
	body := ListDetail(page.List, page.Detail, ratio)
	footer := make([]framework.CanvasObject, 0, 2)
	if page.Status != nil {
		footer = append(footer, page.Status)
	}
	if page.Paging != nil {
		footer = append(footer, page.Paging)
	}
	var bottom framework.CanvasObject
	if len(footer) > 0 {
		bottom = container.NewVBox(footer...)
	}
	return container.NewBorder(container.NewPadded(container.NewVBox(heading...)), bottom, nil, nil, container.NewPadded(body))
}

// StandardPage builds a non-list workflow page with the same heading and
// action hierarchy as StandardListPage.
func StandardPage(title, subtitle string, actions []framework.CanvasObject, body, status framework.CanvasObject) framework.CanvasObject {
	heading := []framework.CanvasObject{widget.NewLabelWithStyle(title, framework.TextAlignLeading, framework.TextStyle{Bold: true})}
	if subtitle != "" {
		label := widget.NewLabel(subtitle)
		label.Wrapping = framework.TextWrapWord
		heading = append(heading, label)
	}
	if bar := ActionBar(actions, nil); bar != nil {
		heading = append(heading, bar)
	}
	return container.NewBorder(container.NewPadded(container.NewVBox(heading...)), status, nil, nil, container.NewPadded(body))
}

// FormPage describes the standard create/edit presentation. Forms always
// scroll independently and keep status plus commit/cancel actions visible.
type FormPage struct {
	Title, Subtitle string
	Fields          framework.CanvasObject
	Status          framework.CanvasObject
	Save, Cancel    *SemanticButton
}

// StandardFormPage builds a consistent edit-form layout.
func StandardFormPage(page FormPage) framework.CanvasObject {
	heading := []framework.CanvasObject{widget.NewLabelWithStyle(page.Title, framework.TextAlignLeading, framework.TextStyle{Bold: true})}
	if page.Subtitle != "" {
		label := widget.NewLabel(page.Subtitle)
		label.Wrapping = framework.TextWrapWord
		heading = append(heading, label)
	}
	if page.Save != nil {
		Primary(page.Save)
	}
	actions := make([]framework.CanvasObject, 0, 2)
	if page.Cancel != nil {
		actions = append(actions, page.Cancel)
	}
	if page.Save != nil {
		actions = append(actions, page.Save)
	}
	footer := []framework.CanvasObject{}
	if page.Status != nil {
		footer = append(footer, page.Status)
	}
	if len(actions) > 0 {
		footer = append(footer, container.NewHBox(layout.NewSpacer(), container.NewHBox(actions...)))
	}
	return container.NewBorder(container.NewPadded(container.NewVBox(heading...)), container.NewPadded(container.NewVBox(footer...)), nil, nil, container.NewPadded(container.NewVScroll(page.Fields)))
}

// ActionBar keeps primary actions leading and secondary/destructive actions
// trailing across every workspace.
func ActionBar(primary, other []framework.CanvasObject) framework.CanvasObject {
	if len(primary) == 0 && len(other) == 0 {
		return nil
	}
	objects := append([]framework.CanvasObject(nil), primary...)
	objects = append(objects, layout.NewSpacer())
	objects = append(objects, other...)
	return container.NewHBox(objects...)
}

// Primary marks the principal action on a page.
func Primary(button *SemanticButton) *SemanticButton {
	button.Importance = widget.HighImportance
	return button
}

// Destructive marks an irreversible action consistently.
func Destructive(button *SemanticButton) *SemanticButton {
	button.Importance = widget.DangerImportance
	return button
}

// EmptyDetail provides a consistent unselected master/detail state.
func EmptyDetail(entity string) framework.CanvasObject {
	return container.NewCenter(widget.NewLabel("Select " + entity + " to view details"))
}
