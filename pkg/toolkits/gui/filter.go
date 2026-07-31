package gui

import (
	"strings"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// FilterOption is one common, human-readable filter and the expression the
// application list API understands.
type FilterOption struct {
	Label      string
	Expression string
}

// FilterPreset describes a compact dropdown shown beside the expression.
type FilterPreset struct {
	ID          string
	Placeholder string
	Options     []FilterOption
}

// FilterSelect is a semantic dropdown used by the shared filter bar.
type FilterSelect struct {
	widget.Select
	id string
}

func (s *FilterSelect) SemanticID() string { return s.id }

// FilterBar is the common filtering shape for list workspaces. The expression
// remains the source of truth; selecting a preset writes its domain expression
// into that entry so the generated query is visible and editable.
type FilterBar struct {
	Content    framework.CanvasObject
	Expression *SemanticEntry
	Apply      *SemanticButton
	Presets    []*FilterSelect
	Advanced   *FilterDisclosure
	clauses    []string
}

// FilterDisclosure is a layout-safe disclosure for uncommon filters. Some
// Accordion renderers update their painted children without invalidating a
// surrounding Border layout, allowing controls to obscure the workspace.
// Showing a concrete VBox child updates minimum size and reflows list/detail.
type FilterDisclosure struct {
	Content *framework.Container
	Toggle  *SemanticButton
	body    framework.CanvasObject
	open    bool
	changed func()
}

func newFilterDisclosure(body framework.CanvasObject) *FilterDisclosure {
	d := &FilterDisclosure{body: body}
	d.Toggle = NewButton("filters.more", "More filters", func() { d.SetOpen(!d.open) })
	d.Toggle.Importance = widget.LowImportance
	body.Hide()
	d.Content = container.NewVBox(d.Toggle, body)
	return d
}

// SetOpen expands or collapses uncommon filters and forces layout reflow.
func (d *FilterDisclosure) SetOpen(open bool) {
	d.open = open
	if open {
		d.body.Show()
		d.Toggle.SetText("Fewer filters")
	} else {
		d.body.Hide()
		d.Toggle.SetText("More filters")
	}
	d.Content.Refresh()
	if d.changed != nil {
		d.changed()
	}
}

func (d *FilterDisclosure) IsOpen() bool { return d.open }

// NewFilterBar builds a single-line expression bar with common presets on the
// right and optional uncommon controls behind a collapsed disclosure.
func NewFilterBar(expressionID, applyID, placeholder, initial string, presets, uncommon []FilterPreset, settings framework.CanvasObject, apply func(string)) *FilterBar {
	bar := &FilterBar{}
	bar.Expression = NewEntry(expressionID)
	bar.Expression.SetPlaceHolder(placeholder)
	bar.Expression.SetText(initial)
	bar.Apply = Primary(NewButton(applyID, "Apply", func() { apply(strings.TrimSpace(bar.Expression.Text)) }))

	bar.clauses = make([]string, len(presets)+len(uncommon))
	buildPreset := func(i int, preset FilterPreset) *FilterSelect {
		selectWidget := &FilterSelect{id: preset.ID}
		labels := make([]string, 0, len(preset.Options))
		options := make(map[string]string, len(preset.Options))
		for _, option := range preset.Options {
			labels = append(labels, option.Label)
			options[option.Label] = option.Expression
		}
		selectWidget.Options = labels
		selectWidget.PlaceHolder = preset.Placeholder
		selectWidget.OnChanged = func(label string) {
			bar.replaceClause(i, options[label])
		}
		selectWidget.ExtendBaseWidget(selectWidget)
		bar.Presets = append(bar.Presets, selectWidget)
		return selectWidget
	}
	common := make([]framework.CanvasObject, 0, len(presets)+1)
	for i, preset := range presets {
		common = append(common, buildPreset(i, preset))
	}
	common = append(common, bar.Apply)
	top := container.NewBorder(nil, nil, nil, container.NewHBox(common...), bar.Expression)
	advancedObjects := make([]framework.CanvasObject, 0, len(uncommon)+1)
	for i, preset := range uncommon {
		advancedObjects = append(advancedObjects, buildPreset(len(presets)+i, preset))
	}
	if settings != nil {
		advancedObjects = append(advancedObjects, settings)
	}
	if len(advancedObjects) == 0 {
		bar.Content = top
		return bar
	}
	bar.Advanced = newFilterDisclosure(container.NewPadded(container.NewVBox(advancedObjects...)))
	bar.Content = container.NewVBox(top, bar.Advanced.Content)
	return bar
}

func (b *FilterBar) replaceClause(index int, clause string) {
	expression := strings.TrimSpace(b.Expression.Text)
	old := b.clauses[index]
	if old != "" {
		parts := strings.Split(expression, " && ")
		kept := parts[:0]
		for _, part := range parts {
			if strings.TrimSpace(part) != old {
				kept = append(kept, part)
			}
		}
		expression = strings.Join(kept, " && ")
	}
	b.clauses[index] = strings.TrimSpace(clause)
	if b.clauses[index] != "" {
		if expression != "" {
			expression += " && "
		}
		expression += b.clauses[index]
	}
	b.Expression.SetText(expression)
}

// SetEnabled changes every interactive part of the filter bar together.
func (b *FilterBar) SetEnabled(enabled bool) {
	controls := []interface {
		Enable()
		Disable()
	}{b.Expression, b.Apply}
	for _, preset := range b.Presets {
		controls = append(controls, preset)
	}
	for _, control := range controls {
		if enabled {
			control.Enable()
		} else {
			control.Disable()
		}
	}
}
