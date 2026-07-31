package gui

import (
	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ListPage describes the standard primary/secondary workspace used by
// catalog-style application surfaces. Domains provide content and commands;
// the toolkit owns visual hierarchy and placement.
type ListPage struct {
	Title, Subtitle   string
	Filters           framework.CanvasObject
	FilterDisclosure  *FilterDisclosure
	CollectionActions []framework.CanvasObject
	OtherActions      []framework.CanvasObject // page-scoped trailing actions, when needed
	DetailActions     []framework.CanvasObject
	List, Detail      framework.CanvasObject
	Status, Paging    framework.CanvasObject
	ListRatio         float64
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
	if actions := ActionBar(page.CollectionActions, page.OtherActions); actions != nil {
		heading = append(heading, actions)
	}
	if page.Filters != nil {
		heading = append(heading, widget.NewSeparator(), page.Filters)
	}
	ratio := page.ListRatio
	if ratio <= 0 || ratio >= 1 {
		ratio = .38
	}
	detail := page.Detail
	if actions := DetailActionBar(page.DetailActions); actions != nil {
		detail = container.NewBorder(container.NewPadded(actions), nil, nil, nil, page.Detail)
	}
	body := ListDetail(page.List, detail, ratio)
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
	root := container.NewBorder(container.NewPadded(container.NewVBox(heading...)), bottom, nil, nil, container.NewPadded(body))
	// Dynamic filter disclosures must invalidate the owning Border layout, not
	// only repaint themselves; otherwise their expanded children can obscure
	// the list/detail workspace.
	if page.FilterDisclosure != nil {
		page.FilterDisclosure.changed = func() {
			root.Layout.Layout(root.Objects, root.Size())
			root.Refresh()
		}
	}
	return root
}

// DetailActionBar keeps actions that operate on the selected entity in its
// own pane and wraps dense action sets into predictable rows.
func DetailActionBar(actions []framework.CanvasObject) framework.CanvasObject {
	if len(actions) == 0 {
		return nil
	}
	columns := min(3, len(actions))
	return container.NewGridWithColumns(columns, actions...)
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
	return EmptyState(theme.InfoIcon(), "Nothing selected", "Select "+entity+" from the list to view details.", nil)
}

// EmptyCollection distinguishes a successful empty query from loading or a
// broken canvas and gives the user a useful next step.
func EmptyCollection(entity, guidance string) framework.CanvasObject {
	if guidance == "" {
		guidance = "Adjust the filters or add the first " + entity + "."
	}
	return EmptyState(theme.ListIcon(), "No "+entity+" found", guidance, nil)
}

// EmptyState renders intentional absence with a symbol, a concise explanation,
// and an optional next action. It is shared by empty collections and detail
// panes so absence never resembles a loading or rendering failure.
func EmptyState(icon framework.Resource, title, guidance string, action framework.CanvasObject) framework.CanvasObject {
	objects := []framework.CanvasObject{widget.NewIcon(icon), widget.NewLabelWithStyle(title, framework.TextAlignCenter, framework.TextStyle{Bold: true})}
	if guidance != "" {
		label := widget.NewLabel(guidance)
		label.Alignment = framework.TextAlignCenter
		label.Wrapping = framework.TextWrapWord
		objects = append(objects, label)
	}
	if action != nil {
		objects = append(objects, container.NewCenter(action))
	}
	return container.NewCenter(container.NewPadded(container.NewVBox(objects...)))
}

// StatusKind is the repeated visual and textual vocabulary for transient and
// terminal feedback. Symbols ensure meaning is never carried by color alone.
type StatusKind uint8

const (
	StatusInformational StatusKind = iota
	StatusLoading
	StatusSuccess
	StatusWarning
	StatusError
)

// StatusLine builds status feedback with a stable symbol and semantic widget
// importance. Loading remains informational because it is not a warning.
func StatusLine(kind StatusKind, message string) framework.CanvasObject {
	icon := theme.InfoIcon()
	importance := widget.MediumImportance
	switch kind {
	case StatusInformational:
		// The initialized icon and importance are the informational treatment.
	case StatusLoading:
		icon = theme.ViewRefreshIcon()
	case StatusSuccess:
		icon = theme.ConfirmIcon()
		importance = widget.SuccessImportance
	case StatusWarning:
		icon = theme.WarningIcon()
		importance = widget.WarningImportance
	case StatusError:
		icon = theme.ErrorIcon()
		importance = widget.DangerImportance
	}
	label := widget.NewLabel(message)
	label.Importance = importance
	label.Wrapping = framework.TextWrapWord
	return container.NewHBox(widget.NewIcon(icon), label)
}
