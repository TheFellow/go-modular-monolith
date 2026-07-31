package gui

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/queries"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

const (
	ControlRefresh           = "menus.refresh"
	ControlCreate            = "menus.create"
	ControlRename            = "menus.rename"
	ControlDelete            = "menus.delete"
	ControlPublish           = "menus.publish"
	ControlDraft             = "menus.draft"
	ControlTags              = "menus.tags"
	ControlAddDrink          = "menus.drink.add"
	ControlAnalyze           = "menus.analyze"
	ControlTargetMargin      = "menus.analysis.target-margin"
	ControlRunAnalysis       = "menus.analysis.run"
	ControlApplyFilter       = "menus.filter.apply"
	ControlFilterStatus      = "menus.filter.status"
	ControlFilterExpression  = "menus.filter.expression"
	ControlPrevious          = "menus.previous"
	ControlNext              = "menus.next"
	ControlName              = "menus.form.name"
	ControlDescription       = "menus.form.description"
	ControlTagValues         = "menus.form.tags"
	ControlDrinkSearch       = "menus.drink.search"
	ControlSave              = "menus.form.save"
	ControlCancel            = "menus.form.cancel"
	ControlBack              = "menus.detail.back"
	ControlBreadcrumb        = "menus.detail.breadcrumb"
	controlRemoveDrinkPrefix = "menus.drink.remove."
	controlDrinkChoicePrefix = "menus.drink.choice."
)

type semanticSelect struct {
	widget.Select
	id string
}

func newSelect(id string, options []string) *semanticSelect {
	s := &semanticSelect{id: id}
	s.Options = options
	s.ExtendBaseWidget(s)
	return s
}
func (s *semanticSelect) SemanticID() string { return s.id }

type View struct {
	p                                                                               *Presenter
	root, browse, detail, drinkPanel, analysisPanel, tagsPanel                      *framework.Container
	list                                                                            *widget.Table
	listStack, empty                                                                *framework.Container
	filterLimit                                                                     *semanticSelect
	filterStatus                                                                    *ui.FilterSelect
	filterExpression, name, description, tags, drinkSearch, drinkTags, targetMargin *ui.SemanticEntry
	status, formStatus, drinkStatus, analysisStatus                                 *widget.Label
	descriptionHelp                                                                 *widget.Label
	save, cancel, refresh, create, previous, next                                   *ui.SemanticButton
	applyFilter                                                                     *ui.SemanticButton
	rename, delete, publish, draft, analyze, tagAction, addDrink                    *ui.SemanticButton
	drinkSearchAction, drinkCancel, runAnalysis, analysisCancel                     *ui.SemanticButton
	drinkChoices                                                                    *framework.Container
	drinkTagPreview                                                                 *ui.TagPreview
	state                                                                           State
	rendering                                                                       bool
	renderedMode                                                                    Mode
	renderedForm                                                                    Form
	renderedInstance                                                                uint64
}

var _ ui.View = (*View)(nil)
var _ ui.Activated = (*View)(nil)

func NewView(p *Presenter) *View {
	v := &View{p: p, state: p.State()}
	v.filterLimit = newSelect("menus.filter.limit", []string{"25", "50", "100"})
	v.filterLimit.SetSelected(strconv.Itoa(v.state.Filter.Limit))
	bar := ui.NewSingleRowFilterBar(ControlFilterExpression, ControlApplyFilter, `Filter menus (for example: name.contains("summer"))`, v.state.Filter.Expression,
		[]ui.FilterPreset{{ID: ControlFilterStatus, Placeholder: "Status", Options: []ui.FilterOption{{Label: "Any status"}, {Label: "Draft", Expression: `status == "draft"`}, {Label: "Published", Expression: `status == "published"`}, {Label: "Archived", Expression: `status == "archived"`}}}},
		container.NewBorder(nil, nil, widget.NewLabel("Page size"), nil, v.filterLimit), func(expression string) {
			limit, _ := strconv.Atoi(v.filterLimit.Selected)
			if p.SetFilter(Filter{Expression: expression, Limit: limit}) {
				p.Refresh()
			}
		})
	v.filterExpression, v.filterStatus = bar.Expression, bar.Presets[0]
	v.applyFilter = bar.Apply
	v.descriptionHelp = widget.NewLabel("")
	v.descriptionHelp.Hide()
	columns := []string{"Name", "Status", "Items", "Created", "Published", "Tags", "Actions"}
	v.list = widget.NewTable(func() (int, int) { return len(v.state.Items) + 1, len(columns) }, func() framework.CanvasObject {
		return ui.NewActionCell()
	}, func(id widget.TableCellID, o framework.CanvasObject) {
		cell := o
		if id.Row == 0 {
			ui.ShowCellText(cell, columns[id.Col], true)
			return
		}
		m := v.state.Items[id.Row-1]
		published := ""
		if t, ok := m.PublishedAt.Unwrap(); ok {
			published = t.Format(time.RFC3339)
		}
		values := []string{m.Name, string(m.Status), strconv.Itoa(len(m.Items)), formatTime(m.CreatedAt), published, m.Tags.Canonical().String()}
		if id.Col == len(columns)-1 {
			index := id.Row - 1
			actions := []ui.RowAction{{Label: "View", Run: func() { p.Select(index) }}}
			canPublish, canDraft := p.ListPermissions(index)
			if canPublish {
				actions = append(actions, ui.RowAction{Label: "Publish", Run: func() { p.Select(index); p.Publish() }})
			}
			if canDraft {
				actions = append(actions, ui.RowAction{Label: "Return to draft", Run: func() { p.Select(index); p.ReturnToDraft() }})
			}
			ui.ShowCellActions(cell, actions)
			return
		}
		ui.ShowCellText(cell, values[id.Col], false)
	})
	v.list.OnSelected = func(id widget.TableCellID) {
		if id.Row > 0 && id.Col < len(columns)-1 {
			v.list.UnselectAll()
			p.Select(id.Row - 1)
		}
	}
	for i, w := range []float32{210, 100, 70, 190, 190, 190, 140} {
		v.list.SetColumnWidth(i, w)
	}
	v.empty = ui.EmptyCollection(ui.IconEmpty, "No menus found", "Adjust the filter or create a menu.")
	v.listStack = container.NewStack(v.list, v.empty)
	v.refresh = ui.WithIcon(ui.NewButton(ControlRefresh, "Refresh", p.Refresh), ui.IconRefresh)
	v.create = ui.Primary(ui.WithIcon(ui.NewButton(ControlCreate, "New menu", p.StartCreate), ui.IconAdd))
	v.previous = ui.WithIcon(ui.NewButton(ControlPrevious, "Previous", p.PreviousPage), ui.IconPrevious)
	v.next = ui.WithIcon(ui.NewButton(ControlNext, "Next", p.NextPage), ui.IconNext)
	v.status = widget.NewLabel("")
	v.browse = ui.StandardListPage(ui.ListPage{Title: "Menus", Subtitle: "Browse menus and select one for complete details.", Filters: bar.Content, CollectionActions: []framework.CanvasObject{v.create, v.refresh}, List: v.listStack, Status: v.status, Paging: container.NewHBox(v.previous, v.next)}).(*framework.Container)

	v.name, v.description, v.tags = ui.NewEntry(ControlName), ui.NewEntry(ControlDescription), ui.NewEntry(ControlTagValues)
	v.description.MultiLine = true
	v.save = ui.WithIcon(ui.NewButton(ControlSave, "Save", func() { v.readForm(); p.Save() }), ui.IconSave)
	v.cancel = ui.WithIcon(ui.NewButton(ControlCancel, "Cancel", p.Cancel), ui.IconCancel)
	v.formStatus = widget.NewLabel("")
	v.rename = ui.NewButton(ControlRename, "Edit", p.StartRename)
	v.rename.Hide()
	v.addDrink = ui.WithIcon(ui.NewButton(ControlAddDrink, "Add drink", p.StartAddDrink), ui.IconAdd)
	v.analyze = ui.NewButton(ControlAnalyze, "Analyze", p.StartAnalysis)
	v.publish = ui.NewButton(ControlPublish, "Publish", p.Publish)
	v.draft = ui.NewButton(ControlDraft, "Return to draft", p.ReturnToDraft)
	v.tagAction = ui.WithIcon(ui.NewButton(ControlTags, "Tags", p.StartTags), ui.IconTag)
	v.delete = ui.Destructive(ui.WithIcon(ui.NewButton(ControlDelete, "Delete", p.Delete), ui.IconDelete))
	v.detail = v.buildDetail(v.state)
	v.tagsPanel = v.buildTags(v.state)

	v.drinkSearch, v.drinkTags = ui.NewEntry(ControlDrinkSearch), ui.NewEntry(ControlTagValues+".add-drink")
	v.drinkTagPreview = ui.NewTagPreview("")
	v.drinkSearch.SetPlaceHolder("Search active drinks")
	v.drinkSearchAction = ui.NewButton(ControlDrinkSearch+".apply", "Search", func() { p.SearchDrinks(v.drinkSearch.Text) })
	v.drinkSearch.OnSubmitted = func(string) { v.drinkSearchAction.OnTapped() }
	v.drinkChoices = container.NewVBox()
	v.drinkStatus = widget.NewLabel("")
	v.drinkCancel = ui.WithIcon(ui.NewButton(ControlCancel+".drink", "Cancel", p.Cancel), ui.IconCancel)
	v.drinkPanel = container.NewBorder(container.NewVBox(v.breadcrumb("Add drink"), widget.NewLabelWithStyle("Add drink", framework.TextAlignLeading, framework.TextStyle{Bold: true}), container.NewBorder(nil, nil, nil, v.drinkSearchAction, v.drinkSearch), field("Tags (complete set)", ui.TagEditor(v.drinkTagPreview, v.drinkTags))), container.NewVBox(v.drinkStatus, container.NewHBox(layout.NewSpacer(), v.drinkCancel)), nil, nil, container.NewVScroll(v.drinkChoices))
	v.targetMargin = ui.NewEntry(ControlTargetMargin)
	v.runAnalysis = ui.NewButton(ControlRunAnalysis, "Analyze", func() { p.SetAnalysisForm(AnalysisForm{TargetMargin: v.targetMargin.Text}); p.Analyze() })
	v.analysisCancel = ui.WithIcon(ui.NewButton(ControlCancel+".analysis", "Close", p.Cancel), ui.IconCancel)
	v.analysisStatus = widget.NewLabel("")
	v.analysisStatus.Wrapping = framework.TextWrapWord
	v.analysisPanel = container.NewBorder(container.NewVBox(v.breadcrumb("Analysis"), widget.NewLabelWithStyle("Menu cost and availability analysis", framework.TextAlignLeading, framework.TextStyle{Bold: true}), field("Target margin (0–1)", v.targetMargin), v.runAnalysis), container.NewHBox(layout.NewSpacer(), v.analysisCancel), nil, nil, container.NewVScroll(v.analysisStatus))
	v.root = container.NewStack(v.browse, v.detail, v.tagsPanel, v.drinkPanel, v.analysisPanel)
	v.name.OnChanged = func(string) { v.changed() }
	v.description.OnChanged = func(string) { v.changed() }
	v.tags.OnChanged = func(string) { v.changed() }
	v.drinkTags.OnChanged = func(value string) { v.drinkTagPreview.SetCSV(value) }
	p.Observe(v.render)
	return v
}

func (v *View) breadcrumb(name string) framework.CanvasObject {
	return container.NewHBox(ui.WithIcon(ui.NewButton(ControlBack, "Back", v.p.Back), ui.IconBack), ui.NewButton(ControlBreadcrumb, "Menus", v.p.ResetList), widget.NewLabel(">"), widget.NewLabel(name))
}
func (v *View) Title() string                   { return "Menus" }
func (v *View) Content() framework.CanvasObject { return v.root }
func (v *View) Activate()                       { v.p.ResetList() }
func (v *View) HasUnsavedChanges() bool {
	s := v.p.State()
	return s.Dirty || s.Mode == Creating || s.Mode == Tagging || s.Mode == AddingDrink
}
func (v *View) ExecuteCommand(c ui.Command) bool {
	s := v.p.State()
	switch c {
	case ui.CommandRefresh:
		return s.Mode == Browsing && ui.Trigger(v.refresh)
	case ui.CommandNew:
		return s.Mode == Browsing && ui.Trigger(v.create)
	case ui.CommandSave:
		if s.Mode == Analyzing {
			return ui.Trigger(v.runAnalysis)
		}
		return ui.Trigger(v.save)
	case ui.CommandCancel:
		return s.Mode != Browsing && ui.Trigger(v.cancel)
	}
	return false
}
func (v *View) changed() {
	if v.rendering {
		return
	}
	s := v.p.State()
	if s.Mode == Viewing {
		v.populate(s.Form)
		return
	}
	if s.Mode != Editing && s.Mode != Renaming && s.Mode != Creating && s.Mode != Tagging {
		return
	}
	v.readForm()
}
func (v *View) readForm() {
	s := v.p.State()
	if s.Mode == Tagging {
		v.p.SetForm(Form{Tags: v.tags.Text})
		return
	}
	v.p.SetForm(Form{Name: v.name.Text, Description: v.description.Text, Tags: v.tags.Text, ReplaceTags: true})
}
func (v *View) populate(f Form) {
	v.rendering = true
	defer func() { v.rendering = false }()
	v.name.SetText(f.Name)
	v.description.SetText(f.Description)
	v.tags.SetText(f.Tags)
}

func (v *View) buildDetail(s State) *framework.Container {
	name := "Menu"
	selected := s.Selected
	if s.Mode == Creating {
		name = "New menu"
		selected = nil
	} else if selected != nil {
		name = selected.Name
	}
	tagValue := ui.TagEditor(ui.NewTagPreview(v.tags.Text), v.tags)
	if s.Mode == Viewing && selected != nil {
		tagValue = ui.TagPillsCSV(selected.Tags.Canonical().String())
	}
	fields := container.NewVBox(field("Name", v.name), field("Description", v.description), field("Tags", tagValue))
	if selected != nil {
		m := selected
		published := ""
		if t, ok := m.PublishedAt.Unwrap(); ok {
			published = t.Format(time.RFC3339)
		}
		meta := ui.DetailForm(ui.DetailField("Status", readonly(string(m.Status))), ui.DetailField("Created", readonly(formatTime(m.CreatedAt))), ui.DetailField("Published", readonly(published)))
		fields.Add(meta)
		fields.Add(widget.NewLabelWithStyle(fmt.Sprintf("Drinks (%d)", len(m.Items)), framework.TextAlignLeading, framework.TextStyle{Bold: true}))
		items := append([]models.MenuItem(nil), m.Items...)
		sort.SliceStable(items, func(i, j int) bool { return items[i].SortOrder < items[j].SortOrder })
		if len(items) == 0 {
			fields.Add(ui.EmptyCollection(ui.IconEmpty, "No drinks on this menu", "Add a drink before publishing."))
		}
		for _, item := range items {
			price := "N/A"
			if p, ok := item.Price.Unwrap(); ok {
				price = p.String()
			}
			line := fmt.Sprintf("%s  ·  %s  ·  %s  ·  order %d", v.p.DrinkName(item.DrinkID), price, item.Availability, item.SortOrder)
			row := []framework.CanvasObject{widget.NewLabel(line)}
			if s.CanRemoveDrink && m.Status == models.MenuStatusDraft && !s.Dirty && !s.Submitting && !s.Confirming {
				row = append(row, ui.WithIcon(ui.NewButton(controlRemoveDrinkPrefix+item.DrinkID.String(), "Remove", func() { v.p.RemoveDrink(item.DrinkID) }), ui.IconDelete))
			}
			fields.Add(container.NewBorder(nil, nil, nil, container.NewHBox(row[1:]...), row[0]))
		}
	}
	actions := []framework.CanvasObject{}
	if selected != nil && (s.Mode == Viewing || s.Mode == Editing) && !s.Dirty && !s.Submitting && !s.Confirming {
		draft := selected.Status == models.MenuStatusDraft
		if s.CanAddDrink && draft {
			actions = append(actions, v.addDrink)
		}
		actions = append(actions, v.analyze)
		if s.CanPublish && draft && len(selected.Items) > 0 {
			actions = append(actions, v.publish)
		}
		if s.CanDraft && selected.Status == models.MenuStatusPublished {
			actions = append(actions, v.draft)
		}
		if s.CanTag {
			actions = append(actions, v.tagAction)
		}
		if s.CanDelete && draft {
			actions = append(actions, v.delete)
		}
	}
	body := []framework.CanvasObject{}
	if bar := ui.ActionBar(nil, actions); bar != nil {
		body = append(body, bar)
	}
	body = append(body, fields)
	// Kept as a hidden semantic target for command/test compatibility. Row
	// selection already opens the editable detail when authorization permits.
	body = append(body, v.rename)
	return ui.StandardFormPage(ui.FormPage{Title: name, Breadcrumb: v.breadcrumb(name), Fields: container.NewVBox(body...), Status: v.formStatus, Save: v.save, Cancel: v.cancel}).(*framework.Container)
}

func (v *View) buildTags(s State) *framework.Container {
	name := "Menu"
	if s.Selected != nil {
		name = s.Selected.Name
	}
	return ui.StandardFormPage(ui.FormPage{Title: "Edit tags", Breadcrumb: v.breadcrumb(name), Fields: container.NewVBox(field("Tags (complete set)", ui.TagEditor(ui.NewTagPreview(v.tags.Text), v.tags))), Status: v.formStatus, Save: v.save, Cancel: v.cancel}).(*framework.Container)
}

func (v *View) render(s State) {
	v.state = s
	if s.Mode == AddingDrink && v.renderedMode != AddingDrink {
		v.drinkSearch.SetText(s.Form.DrinkQuery)
		v.drinkTags.SetText(s.Form.Tags)
	}
	if s.Mode == Analyzing && v.renderedMode != Analyzing {
		v.targetMargin.SetText(s.AnalysisForm.TargetMargin)
	}
	if (s.Mode == Editing || s.Mode == Viewing || s.Mode == Creating || s.Mode == Tagging) && (v.renderedMode != s.Mode || v.renderedInstance != s.FormInstance || !reflect.DeepEqual(v.renderedForm, s.Form)) {
		v.populate(s.Form)
		v.renderedMode, v.renderedInstance, v.renderedForm = s.Mode, s.FormInstance, s.Form
	}
	busy := s.Loading || s.Submitting || s.Confirming
	cleanDetail := s.Selected != nil && (s.Mode == Viewing || s.Mode == Editing) && !s.Dirty && !busy
	draftMenu := cleanDetail && s.Selected.Status == models.MenuStatusDraft
	v.addDrink.Hidden = !draftMenu || !s.CanAddDrink
	v.analyze.Hidden = !cleanDetail
	v.publish.Hidden = !draftMenu || !s.CanPublish || len(s.Selected.Items) == 0
	v.draft.Hidden = !cleanDetail || !s.CanDraft || s.Selected.Status != models.MenuStatusPublished
	v.tagAction.Hidden = !cleanDetail || !s.CanTag
	v.delete.Hidden = !draftMenu || !s.CanDelete
	v.descriptionHelp.Hidden = s.Mode != Editing && s.Mode != Renaming
	setEnabled(v.previous, !busy && len(s.History) > 0)
	setEnabled(v.next, !busy && s.Next != "")
	setEnabled(v.refresh, !busy)
	setEnabled(v.create, !busy)
	v.create.Hidden = !s.CanCreate
	setEnabled(v.filterExpression, !busy)
	setEnabled(v.filterStatus, !busy)
	setEnabled(v.filterLimit, !busy)
	setEnabled(v.applyFilter, !busy)
	setEnabled(v.rename, !busy)
	setEnabled(v.tagAction, !busy)
	setEnabled(v.delete, !busy)
	v.empty.Hidden = s.Loading || len(s.Items) > 0
	v.list.Hidden = !v.empty.Hidden
	if s.Loading {
		v.status.SetText("Loading menus…")
	} else if s.Err != nil {
		v.status.SetText("Error: " + s.Err.Error())
	} else {
		v.status.SetText(fmt.Sprintf("%d menus", len(s.Items)))
	}
	if s.Submitting {
		v.formStatus.SetText("Saving…")
	} else if s.Err != nil {
		v.formStatus.SetText("Error: " + s.Err.Error())
	} else {
		v.formStatus.SetText("")
	}
	canSave := !busy && (s.Mode == Creating || (s.Mode == Editing && s.Dirty) || (s.Mode == Renaming && s.Dirty) || (s.Mode == Tagging && s.Dirty))
	setEnabled(v.save, canSave)
	setEnabled(v.cancel, canSave || s.Mode == Creating || s.Mode == Tagging)
	setEnabled(v.name, !busy)
	setEnabled(v.description, !busy)
	setEnabled(v.tags, !busy)
	v.drinkChoices.RemoveAll()
	for _, option := range s.Drinks {
		option := option
		b := ui.NewButton(controlDrinkChoicePrefix+option.ID.String(), option.Name, func() {
			f := v.p.State().Form
			f.Tags, f.ReplaceTags = v.drinkTags.Text, true
			v.p.SetForm(f)
			v.p.AddDrink(option.ID)
		})
		if busy {
			b.Disable()
		}
		v.drinkChoices.Add(b)
	}
	if len(s.Drinks) == 0 && s.Mode == AddingDrink && !s.Loading {
		v.drinkChoices.Add(ui.EmptyCollection(ui.IconEmpty, "No matching drinks", "Try another search."))
	}
	msg := ""
	if s.Loading {
		msg = "Loading…"
	} else if s.Err != nil {
		msg = "Error: " + s.Err.Error()
	}
	v.drinkStatus.SetText(msg)
	if s.Analysis != nil {
		v.analysisStatus.SetText(analysisText(*s.Analysis))
	} else {
		v.analysisStatus.SetText(msg)
	}
	v.detail = v.buildDetail(s)
	v.tagsPanel = v.buildTags(s)
	v.renderedMode = s.Mode
	v.browse.Hidden = s.Mode != Browsing
	v.detail.Hidden = s.Mode != Viewing && s.Mode != Editing && s.Mode != Creating && s.Mode != Renaming
	v.tagsPanel.Hidden = s.Mode != Tagging
	v.drinkPanel.Hidden = s.Mode != AddingDrink
	v.analysisPanel.Hidden = s.Mode != Analyzing
	switch s.Mode {
	case Browsing:
		v.root.Objects = []framework.CanvasObject{v.browse, v.detail, v.tagsPanel, v.drinkPanel, v.analysisPanel}
	case Viewing, Editing, Creating:
		v.root.Objects = []framework.CanvasObject{v.browse, v.detail, v.tagsPanel, v.drinkPanel, v.analysisPanel}
	case AddingDrink:
		v.root.Objects = []framework.CanvasObject{v.browse, v.detail, v.tagsPanel, v.drinkPanel, v.analysisPanel}
	case Analyzing:
		v.root.Objects = []framework.CanvasObject{v.browse, v.detail, v.tagsPanel, v.drinkPanel, v.analysisPanel}
	case Tagging:
		v.root.Objects = []framework.CanvasObject{v.browse, v.detail, v.tagsPanel, v.drinkPanel, v.analysisPanel}
	case Renaming:
		v.root.Objects = []framework.CanvasObject{v.browse, v.detail, v.tagsPanel, v.drinkPanel, v.analysisPanel}
	}
	v.list.Refresh()
	v.root.Refresh()
}

func analysisText(a queries.MenuAnalytics) string {
	lines := []string{fmt.Sprintf("Available: %d/%d", a.AvailableCount, a.TotalCount)}
	if a.AverageMargin != nil {
		lines = append(lines, fmt.Sprintf("Average margin: %.0f%%", *a.AverageMargin*100))
	}
	for _, i := range a.Items {
		cost, price, margin := "n/a", "n/a", "n/a"
		if i.Cost != nil && !i.CostUnknown {
			cost = i.Cost.String()
		}
		if i.MenuPrice != nil {
			price = i.MenuPrice.String()
		} else if i.SuggestedPrice != nil {
			price = "suggested " + i.SuggestedPrice.String()
		}
		if i.Margin != nil {
			margin = fmt.Sprintf("%.0f%%", *i.Margin*100)
		}
		lines = append(lines, fmt.Sprintf("\n%s\nCost: %s\nPrice: %s\nMargin: %s\nStatus: %s", i.Name, cost, price, margin, strings.ToUpper(string(i.Availability))))
	}
	return strings.Join(lines, "\n")
}
func readonly(value string) *widget.Entry {
	e := widget.NewEntry()
	restoring := false
	e.OnChanged = func(changed string) {
		if restoring || changed == value {
			return
		}
		restoring = true
		e.SetText(value)
		restoring = false
	}
	e.SetText(value)
	return e
}
func setEnabled(o interface {
	Enable()
	Disable()
}, enabled bool) {
	if enabled {
		o.Enable()
	} else {
		o.Disable()
	}
}
func field(label string, o framework.CanvasObject) framework.CanvasObject {
	return ui.DetailField(label, o)
}
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
func detailText(menu *models.Menu, drinkName func(entity.DrinkID) string) string {
	if menu == nil {
		return ""
	}
	parts := []string{menu.Name, "Created: " + formatTime(menu.CreatedAt), "Status: " + string(menu.Status), "Tags: " + menu.Tags.Canonical().String()}
	if published, ok := menu.PublishedAt.Unwrap(); ok {
		parts = append(parts, "Published: "+formatTime(published))
	}
	items := append([]models.MenuItem(nil), menu.Items...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].SortOrder < items[j].SortOrder })
	for _, item := range items {
		price := "N/A"
		if value, ok := item.Price.Unwrap(); ok {
			price = value.String()
		}
		featured := ""
		if item.Featured {
			featured = "\nfeatured"
		}
		parts = append(parts, fmt.Sprintf("%s\nDrink ID: %s\nSort order: %d\n%s\n%s%s", drinkName(item.DrinkID), item.DrinkID.String(), item.SortOrder, item.Availability, price, featured))
	}
	return strings.Join(parts, "\n")
}
