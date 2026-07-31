package gui

import (
	"fmt"
	"reflect"
	"strconv"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

const (
	ControlFilter       = "ingredients-filter"
	ControlApplyFilter  = "ingredients-apply-filter"
	ControlRefresh      = "ingredients-refresh"
	ControlCreate       = "ingredients-create"
	ControlEdit         = "ingredients-edit"
	ControlDelete       = "ingredients-delete"
	ControlTags         = "ingredients-tags"
	ControlPrevious     = "ingredients-previous"
	ControlNext         = "ingredients-next"
	ControlSelectPrefix = "ingredient-select-"
	ControlFormTags     = "ingredient-form-tags"
	ControlName         = "ingredient-form-name"
	ControlDescription  = "ingredient-form-description"
	ControlMutationTags = "ingredient-form-mutation-tags"
	ControlSave         = "ingredient-form-save"
	ControlCancel       = "ingredient-form-cancel"
	ControlBack         = "ingredient-detail-back"
	ControlBreadcrumb   = "ingredient-detail-breadcrumb"
)

type View struct {
	presenter                                     *Presenter
	root, browse, formPanel, tagsPanel            *framework.Container
	list                                          *widget.Table
	listStack                                     *framework.Container
	empty                                         *framework.Container
	expression                                    *ui.SemanticEntry
	limit, formCategory, formUnit                 *widget.Select
	name, description                             *ui.SemanticEntry
	tags, tagOnly                                 *ui.TagTokenEditor
	save, cancel, refresh, create, previous, next *ui.SemanticButton
	tagSave, tagCancel                            *ui.SemanticButton
	tagAction, delete                             *ui.SemanticButton
	status, formStatus, detailTitle, crumbName    *widget.Label
	tagStatus                                     *widget.Label
	rendering                                     bool
	renderedMode                                  Mode
	renderedForm                                  Form
	renderedInstance                              uint64
	state                                         State
}

var _ ui.View = (*View)(nil)
var _ ui.Activated = (*View)(nil)

func NewView(p *Presenter) *View {
	v := &View{presenter: p}
	v.limit = widget.NewSelect([]string{"25", "50", "100"}, nil)
	v.limit.SetSelected(strconv.Itoa(p.Snapshot().Limit))
	presets := []ui.FilterOption{{Label: "Any category"}}
	for _, category := range models.AllCategories() {
		presets = append(presets, ui.FilterOption{Label: string(category), Expression: fmt.Sprintf(`category == %q`, category)})
	}
	bar := ui.NewSingleRowFilterBar(ControlFilter, ControlApplyFilter, `Filter ingredients (for example: name.contains("gin"))`, p.Snapshot().Expression,
		[]ui.FilterPreset{{ID: "ingredients-filter-category", Placeholder: "Category", Options: presets}},
		container.NewBorder(nil, nil, widget.NewLabel("Page size"), nil, v.limit), func(expression string) {
			limit, _ := strconv.Atoi(v.limit.Selected)
			p.Filter("", expression, limit)
		})
	v.expression = bar.Expression
	v.state = p.Snapshot()
	columns := []string{"Name", "Category", "Unit", "Description", "Tags", "Actions"}
	v.list = ui.NewRowTable(func() (int, int) { return len(v.state.Items) + 1, len(columns) }, func() framework.CanvasObject {
		return ui.NewActionCell()
	}, func(id widget.TableCellID, object framework.CanvasObject) {
		cell := object
		if id.Row == 0 {
			ui.ShowCellText(cell, columns[id.Col], true)
			return
		}
		item := v.state.Items[id.Row-1]
		values := []string{item.Name, string(item.Category), string(item.Unit), item.Description, item.Tags.Canonical().String()}
		if id.Col == len(columns)-1 {
			itemID := item.ID
			ui.ShowCellActions(cell, []ui.RowAction{{Label: "View", Run: func() { p.Select(itemID) }}})
			return
		}
		ui.ShowCellText(cell, values[id.Col], false)
	})
	v.list.OnSelected = func(id widget.TableCellID) {
		if id.Row > 0 && id.Col < len(columns)-1 {
			v.list.UnselectAll()
			p.Select(v.state.Items[id.Row-1].ID)
		}
	}
	for i, width := range []float32{180, 110, 85, 260, 180} {
		v.list.SetColumnWidth(i, width)
	}
	v.list.SetColumnWidth(5, 120)
	v.empty = ui.EmptyCollection(ui.IconEmpty, "No ingredients found", "Adjust the filter or create a new ingredient.")
	v.listStack = container.NewStack(v.list, v.empty)
	v.refresh = ui.WithIcon(ui.NewButton(ControlRefresh, "Refresh", p.Load), ui.IconRefresh)
	v.create = ui.Primary(ui.WithIcon(ui.NewButton(ControlCreate, "New ingredient", p.StartCreate), ui.IconAdd))
	v.previous = ui.WithIcon(ui.NewButton(ControlPrevious, "Previous", p.PreviousPage), ui.IconPrevious)
	v.next = ui.WithIcon(ui.NewButton(ControlNext, "Next", p.NextPage), ui.IconNext)
	v.tagAction = ui.WithIcon(ui.NewButton(ControlTags, "Tags", p.StartTags), ui.IconTag)
	v.delete = ui.Destructive(ui.WithIcon(ui.NewButton(ControlDelete, "Delete", p.RequestDelete), ui.IconDelete))
	v.status = widget.NewLabel("")
	v.browse = ui.StandardListPage(ui.ListPage{Title: "Ingredients", Filters: bar.Content, CollectionActions: []framework.CanvasObject{v.create, v.refresh}, List: v.listStack, Status: v.status, Paging: container.NewHBox(v.previous, v.next), ListRatio: .35}).(*framework.Container)

	v.name = ui.NewEntry(ControlName)
	v.description = ui.NewEntry(ControlDescription)
	v.description.MultiLine = true
	v.tags = ui.NewTagTokenEditor(ControlMutationTags, "")
	v.tags.Normalize = tag.UpsertCollection
	categories := make([]string, 0, len(models.AllCategories()))
	for _, x := range models.AllCategories() {
		categories = append(categories, string(x))
	}
	v.formCategory = widget.NewSelect(categories, nil)
	units := make([]string, 0, len(measurement.AllUnits()))
	for _, x := range measurement.AllUnits() {
		units = append(units, string(x))
	}
	v.formUnit = widget.NewSelect(units, nil)
	v.save = ui.WithIcon(ui.NewButton(ControlSave, "Save", func() { v.readForm(); p.Submit(p.Snapshot().Form) }), ui.IconSave)
	v.cancel = ui.WithIcon(ui.NewButton(ControlCancel, "Cancel", p.Cancel), ui.IconCancel)
	v.detailTitle = widget.NewLabel("Ingredient")
	v.crumbName = widget.NewLabel("")
	v.formStatus = widget.NewLabel("")
	fields := ui.DetailForm(ui.DetailField("Name", v.name), ui.DetailField("Category", v.formCategory), ui.DetailField("Unit", v.formUnit), ui.DetailField("Description", v.description), ui.DetailField("Tags", v.tags.Content))
	breadcrumb := container.NewHBox(ui.WithIcon(ui.NewButton(ControlBack, "Back", p.Back), ui.IconBack), ui.NewButton(ControlBreadcrumb, "Ingredients", p.ResetList), widget.NewLabel(">"), v.crumbName, v.tagAction, v.delete)
	v.formPanel = ui.StandardFormPage(ui.FormPage{TitleLabel: v.detailTitle, Breadcrumb: breadcrumb, Fields: fields, Status: v.formStatus, Save: v.save, Cancel: v.cancel}).(*framework.Container)
	v.tagOnly = ui.NewTagTokenEditor(ControlFormTags, "")
	v.tagOnly.Normalize = tag.UpsertCollection
	v.tagSave = ui.WithIcon(ui.NewButton(ControlSave+".tags", "Save", func() { p.Submit(Form{Tags: v.tagOnly.CSV()}) }), ui.IconSave)
	v.tagCancel = ui.WithIcon(ui.NewButton(ControlCancel+".tags", "Cancel", p.Cancel), ui.IconCancel)
	v.tagStatus = widget.NewLabel("")
	v.tagsPanel = ui.StandardFormPage(ui.FormPage{Title: "Edit ingredient tags", Subtitle: "Type a key or key=value and press Enter.", Fields: v.tagOnly.Content, Status: v.tagStatus, Save: v.tagSave, Cancel: v.tagCancel}).(*framework.Container)
	v.root = container.NewStack(v.browse, v.formPanel, v.tagsPanel)
	v.name.OnChanged = func(string) { v.changed() }
	v.formCategory.OnChanged = func(string) { v.changed() }
	v.formUnit.OnChanged = func(string) { v.changed() }
	v.description.OnChanged = func(string) { v.changed() }
	v.tags.OnChanged = func(string) { v.changed() }
	p.OnChange(v.render)
	v.render(p.Snapshot())
	return v
}

func (v *View) Title() string                   { return "Ingredients" }
func (v *View) Content() framework.CanvasObject { return v.root }
func (v *View) Activate()                       { v.presenter.ResetList() }
func (v *View) HasUnsavedChanges() bool {
	s := v.presenter.Snapshot()
	return s.Dirty || s.Mode == Create || s.Mode == Tags
}
func (v *View) ExecuteCommand(c ui.Command) bool {
	s := v.presenter.Snapshot()
	switch c {
	case ui.CommandRefresh:
		return s.Mode == Browse && ui.Trigger(v.refresh)
	case ui.CommandNew:
		return s.Mode == Browse && ui.Trigger(v.create)
	case ui.CommandSave:
		if s.Mode == Tags {
			return ui.Trigger(v.tagSave)
		}
		return ui.Trigger(v.save)
	case ui.CommandCancel:
		if s.Mode == Tags {
			return ui.Trigger(v.tagCancel)
		}
		return s.Mode != Browse && ui.Trigger(v.cancel)
	}
	return false
}

func (v *View) changed() {
	if v.rendering {
		return
	}
	s := v.presenter.Snapshot()
	if s.Mode == Viewing {
		v.populate(s.Form)
		return
	}
	if s.Mode != Edit && s.Mode != Create {
		return
	}
	v.readForm()
}
func (v *View) readForm() {
	v.presenter.SetForm(Form{Name: v.name.Text, Category: models.Category(v.formCategory.Selected), Unit: measurement.Unit(v.formUnit.Selected), Description: v.description.Text, Tags: v.tags.CSV(), ReplaceTags: true})
}
func (v *View) populate(f Form) {
	v.rendering = true
	defer func() { v.rendering = false }()
	v.name.SetText(f.Name)
	v.formCategory.SetSelected(string(f.Category))
	v.formUnit.SetSelected(string(f.Unit))
	v.description.SetText(f.Description)
	v.tags.SetCSV(f.Tags)
}

func (v *View) render(s State) {
	v.rendering = true
	defer func() { v.rendering = false }()
	v.state = s
	v.browse.Hidden = s.Mode != Browse
	v.formPanel.Hidden = s.Mode != Edit && s.Mode != Viewing && s.Mode != Create
	v.tagsPanel.Hidden = s.Mode != Tags
	if (s.Mode == Edit || s.Mode == Viewing || s.Mode == Create) && (v.renderedMode != s.Mode || v.renderedInstance != s.FormInstance || !reflect.DeepEqual(v.renderedForm, s.Form)) {
		v.populate(s.Form)
		v.renderedMode = s.Mode
		v.renderedInstance = s.FormInstance
		v.renderedForm = s.Form
	}
	if s.Mode == Tags {
		v.tagOnly.SetCSV(s.Form.Tags)
	}
	if len(s.History) == 0 || s.Status == ui.Loading {
		v.previous.Disable()
	} else {
		v.previous.Enable()
	}
	if s.Next == "" || s.Status == ui.Loading {
		v.next.Disable()
	} else {
		v.next.Enable()
	}
	if s.Submitting || (s.Mode == Edit && !s.Dirty) || s.Mode == Viewing {
		v.save.Disable()
		v.cancel.Disable()
	} else {
		v.save.Enable()
		v.cancel.Enable()
	}
	if s.Mode == Viewing {
		v.tags.SetCSV(s.Form.Tags)
		v.tags.SetEnabled(false)
		v.save.Hide()
		v.cancel.Hide()
	} else {
		v.tags.SetEnabled(!s.Submitting)
		v.save.Show()
		v.cancel.Show()
	}
	v.create.Hidden = !s.CanCreate
	v.tagAction.Hidden = s.Selected == nil || !s.CanTag || s.Mode == Create
	v.delete.Hidden = s.Selected == nil || !s.CanDelete || s.Mode == Create
	v.empty.Hidden = s.Status != ui.Loaded || len(s.Items) != 0
	v.list.Hidden = s.Status == ui.Loaded && len(s.Items) == 0
	if s.Selected != nil {
		v.detailTitle.SetText(s.Selected.Name)
		v.crumbName.SetText(s.Selected.Name)
	} else if s.Mode == Create {
		v.detailTitle.SetText("New ingredient")
		v.crumbName.SetText("New")
	}
	switch {
	case s.Status == ui.Loading:
		v.status.SetText("Loading ingredients…")
	case s.Err != nil:
		v.status.SetText("Error: " + s.Err.Error())
	default:
		v.status.SetText(fmt.Sprintf("%d ingredients", len(s.Items)))
	}
	if s.Submitting {
		v.formStatus.SetText("Saving…")
		v.tagStatus.SetText("Saving…")
	} else if s.Err != nil {
		v.formStatus.SetText("Error: " + s.Err.Error())
		v.tagStatus.SetText("Error: " + s.Err.Error())
	} else {
		v.formStatus.SetText("")
		v.tagStatus.SetText("")
	}
	if s.Submitting {
		v.tagSave.Disable()
		v.tagCancel.Disable()
	} else {
		v.tagSave.Enable()
		v.tagCancel.Enable()
	}
	v.list.Refresh()
	v.root.Refresh()
}
