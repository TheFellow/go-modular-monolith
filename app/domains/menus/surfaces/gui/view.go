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

	menusdomain "github.com/TheFellow/go-modular-monolith/app/domains/menus"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/queries"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
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

type View struct {
	p                                                              *Presenter
	root, browse, detail, drinkPanel, analysisPanel, tagsPanel     *framework.Container
	list                                                           *ui.RowTable
	listStack, empty                                               *framework.Container
	filterStatus                                                   *ui.FilterSelect
	filterExpression, name, description, drinkSearch, targetMargin *ui.SemanticEntry
	tags, drinkTags                                                *ui.TagTokenEditor
	status, formStatus, drinkStatus, analysisStatus                *widget.Label
	descriptionHelp                                                *widget.Label
	save, cancel, refresh, create                                  *ui.SemanticButton
	applyFilter                                                    *ui.SemanticButton
	rename, delete, publish, draft, analyze, tagAction, addDrink   *ui.SemanticButton
	drinkSearchAction, drinkCancel, runAnalysis, analysisCancel    *ui.SemanticButton
	drinkChoices                                                   *framework.Container
	state                                                          State
	rendering                                                      bool
	renderedMode                                                   Mode
	renderedForm                                                   Form
	renderedInstance                                               uint64
}

var _ ui.View = (*View)(nil)
var _ ui.Activated = (*View)(nil)

func NewView(p *Presenter) *View {
	v := &View{p: p, state: p.State()}
	bar := ui.NewSingleRowFilterBar(ControlFilterExpression, ControlApplyFilter, `Filter menus (for example: name.contains("summer"))`, v.state.Filter.Expression,
		[]ui.FilterPreset{{ID: ControlFilterStatus, Placeholder: "Status", Options: []ui.FilterOption{{Label: "Any status"}, {Label: "Draft", Expression: `status == "draft"`}, {Label: "Published", Expression: `status == "published"`}, {Label: "Archived", Expression: `status == "archived"`}}}},
		nil, func(expression string) {
			if p.SetFilter(Filter{Expression: expression, Limit: ui.PageLimit}) {
				p.Refresh()
			}
		})
	v.filterExpression, v.filterStatus = bar.Expression, bar.Presets[0]
	v.applyFilter = bar.Apply
	v.descriptionHelp = widget.NewLabel("")
	v.descriptionHelp.Hide()
	columns := []string{"Name", "Status", "Items", "Published", "Tags", "Actions"}
	v.list = ui.NewAutoPagingRowTable(func() (int, int) { return len(v.state.Items), len(columns) }, func() framework.CanvasObject {
		return ui.NewActionCell()
	}, func(id widget.TableCellID, o framework.CanvasObject) {
		cell := o
		m := v.state.Items[id.Row]
		published := ""
		if t, ok := m.PublishedAt.Unwrap(); ok {
			published = ui.TableTimestamp(t)
		}
		values := []string{m.Name, string(m.Status), strconv.Itoa(len(m.Items)), published, m.Tags.Canonical().String()}
		if id.Col == len(columns)-1 {
			index := id.Row
			actions := []ui.RowAction{{Label: "View", Run: func() { p.Select(index) }}}
			projected, err := p.ListActions(index)
			if err != nil {
				p.recordProjectionError(err)
				ui.ShowCellActions(cell, actions)
				return
			}
			if state := projected[menusdomain.ControlPublish]; state.Visible && state.Enabled {
				actions = append(actions, ui.RowAction{Label: "Publish", Run: func() { p.Select(index); p.Publish() }})
			}
			if state := projected[menusdomain.ControlDraft]; state.Visible && state.Enabled {
				actions = append(actions, ui.RowAction{Label: "Return to draft", Run: func() { p.Select(index); p.ReturnToDraft() }})
			}
			ui.ShowCellActions(cell, actions)
			return
		}
		if id.Col == 4 {
			ui.ShowCellTags(cell, values[id.Col])
			return
		}
		ui.ShowCellText(cell, values[id.Col], false)
	}, func() { framework.Do(p.NextPage) })
	v.list.OnSelected = func(id widget.TableCellID) {
		if id.Row >= 0 && id.Col < len(columns)-1 {
			v.list.UnselectAll()
			p.Select(id.Row)
		}
	}
	ui.ConfigureRowTable(v.list, []ui.TableColumn{{Title: "Name", Width: 185, Flex: 2, Sortable: true}, {Title: "Status", Width: 85, Flex: 1, Sortable: true}, {Title: "Items", Width: 60, Flex: 1, Sortable: true}, {Title: "Published", Width: 145, Flex: 1, Sortable: true}, {Title: "Tags", Width: 145, Flex: 2, Sortable: true}, {Title: "Actions", Width: ui.RowActionsWidth}}, func(column int, direction ui.SortDirection) {
		sortColumns := []int{0, 1, 2, 4, 5}
		p.SortItems(sortColumns[column], direction)
	})
	v.empty = ui.EmptyCollection(ui.IconEmpty, "No menus found", "Adjust the filter or create a menu.")
	v.listStack = container.NewStack(v.list, v.empty)
	v.refresh = ui.WithIcon(ui.NewButton(ControlRefresh, "Refresh", p.Refresh), ui.IconRefresh)
	v.create = ui.Primary(ui.WithIcon(ui.NewButton(ControlCreate, "New menu", p.StartCreate), ui.IconAdd))
	v.status = widget.NewLabel("")
	v.browse = ui.StandardListPage(ui.ListPage{Title: "Menus", Subtitle: "Browse menus and select one for complete details.", Filters: bar.Content, CollectionActions: []framework.CanvasObject{v.create, v.refresh}, List: v.listStack, Status: v.status}).(*framework.Container)

	v.name, v.description = ui.NewEntry(ControlName), ui.NewMultiLineEntry(ControlDescription)
	v.tags = ui.NewTagTokenEditor(ControlTagValues, "")
	v.tags.Normalize = tag.UpsertCollection
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

	v.drinkSearch = ui.NewEntry(ControlDrinkSearch)
	v.drinkTags = ui.NewTagTokenEditor(ControlTagValues+".add-drink", "")
	v.drinkTags.Normalize = tag.UpsertCollection
	v.drinkSearch.SetPlaceHolder("Search active drinks")
	v.drinkSearchAction = ui.NewButton(ControlDrinkSearch+".apply", "Search", func() { p.SearchDrinks(v.drinkSearch.Text) })
	ui.SubmitOnEnter(v.drinkSearch, v.drinkSearchAction)
	v.drinkChoices = container.NewVBox()
	v.drinkStatus = widget.NewLabel("")
	v.drinkCancel = ui.WithIcon(ui.NewButton(ControlCancel+".drink", "Cancel", p.Cancel), ui.IconCancel)
	v.drinkPanel = container.NewBorder(container.NewVBox(v.breadcrumb("Add drink"), widget.NewLabelWithStyle("Add drink", framework.TextAlignLeading, framework.TextStyle{Bold: true}), container.NewBorder(nil, nil, nil, v.drinkSearchAction, v.drinkSearch), field("Tags", v.drinkTags.Content)), container.NewVBox(v.drinkStatus, container.NewHBox(layout.NewSpacer(), v.drinkCancel)), nil, nil, container.NewVScroll(v.drinkChoices))
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
	p.Observe(v.render)
	return v
}

func (v *View) breadcrumb(name string) framework.CanvasObject {
	return container.NewHBox(ui.WithIcon(ui.NewButton(ControlBack, "Back", v.p.Back), ui.IconBack), ui.NewButton(ControlBreadcrumb, "Menus", v.p.ResetList), widget.NewLabel("›"), widget.NewLabel(name))
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
		v.p.SetForm(Form{Tags: v.tags.CSV()})
		return
	}
	v.p.SetForm(Form{Name: v.name.Text, Description: v.description.Text, Tags: v.tags.CSV(), ReplaceTags: true})
}
func (v *View) populate(f Form) {
	v.rendering = true
	defer func() { v.rendering = false }()
	v.name.SetText(f.Name)
	v.description.SetText(f.Description)
	v.tags.SetCSV(f.Tags)
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
	var tagValue framework.CanvasObject = v.tags.Content
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
		if s.Readiness != nil {
			if len(s.Readiness.Findings) == 0 {
				fields.Add(widget.NewLabel("Readiness: ready"))
			} else {
				lines := []string{"Readiness:"}
				for _, finding := range s.Readiness.Findings {
					lines = append(lines, fmt.Sprintf("• %s: %s", finding.Severity, finding.Message))
				}
				label := widget.NewLabel(strings.Join(lines, "\n"))
				label.Wrapping = framework.TextWrapWord
				fields.Add(label)
			}
		}
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
			name := widget.NewLabelWithStyle(v.p.DrinkName(item.DrinkID), framework.TextAlignLeading, framework.TextStyle{Bold: true})
			meta := widget.NewLabel(fmt.Sprintf("%s  ·  %s  ·  order %d", price, item.Availability, item.SortOrder))
			removeState := s.Actions[menusdomain.ControlRemoveDrink]
			canRemove := removeState.Visible && removeState.Enabled && !s.Dirty && !s.Submitting && !s.Confirming
			options := []string(nil)
			if canRemove {
				options = []string{"Remove"}
			}
			item := item
			removeTarget := ui.NewButton(controlRemoveDrinkPrefix+item.DrinkID.String(), "Remove", func() { v.p.RemoveDrink(item.DrinkID) })
			removeTarget.Hide() // compatibility/shortcut target; the visible affordance is the compact action menu.
			actions := ui.NewActionSelect(options, func(choice string) {
				if choice == "Remove" {
					v.p.RemoveDrink(item.DrinkID)
				}
			})
			if !canRemove {
				actions.Hide()
			}
			copyID := widget.NewButtonWithIcon("", ui.IconResource(ui.IconCopy), func() {
				if app := framework.CurrentApp(); app != nil {
					app.Clipboard().SetContent(item.DrinkID.String())
				}
			})
			copyID.Importance = widget.LowImportance
			if s.Submitting || s.Confirming {
				actions.Disable()
				copyID.Disable()
			}
			line := container.NewBorder(nil, nil, nil, meta, name)
			trailing := container.NewCenter(container.NewHBox(copyID, actions))
			fields.Add(container.NewVBox(container.NewBorder(nil, nil, nil, trailing, line), removeTarget, widget.NewSeparator()))
		}
	}
	actions := []framework.CanvasObject{}
	if selected != nil && (s.Mode == Viewing || s.Mode == Editing) {
		if actionVisible(s, menusdomain.ControlAddDrink) {
			actions = append(actions, v.addDrink)
		}
		actions = append(actions, v.analyze)
		if actionVisible(s, menusdomain.ControlPublish) {
			actions = append(actions, v.publish)
		}
		if actionVisible(s, menusdomain.ControlDraft) {
			actions = append(actions, v.draft)
		}
		if actionVisible(s, menusdomain.ControlTags) {
			actions = append(actions, v.tagAction)
		}
		if actionVisible(s, menusdomain.ControlDelete) {
			actions = append(actions, v.delete)
		}
		transientReady := !s.Dirty && !s.Loading && !s.Submitting && !s.Confirming
		setEnabled(v.addDrink, transientReady && actionEnabled(s, menusdomain.ControlAddDrink))
		setEnabled(v.publish, transientReady && actionEnabled(s, menusdomain.ControlPublish))
		setEnabled(v.draft, transientReady && actionEnabled(s, menusdomain.ControlDraft))
		setEnabled(v.tagAction, transientReady && actionEnabled(s, menusdomain.ControlTags))
		setEnabled(v.delete, transientReady && actionEnabled(s, menusdomain.ControlDelete))
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
	return ui.StandardFormPage(ui.FormPage{Title: "Edit tags", Breadcrumb: v.breadcrumb(name), Subtitle: "Type a key or key=value and press Enter.", Fields: container.NewVBox(field("Tags", v.tags.Content)), Status: v.formStatus, Save: v.save, Cancel: v.cancel}).(*framework.Container)
}

func (v *View) render(s State) {
	v.state = s
	if s.Mode == AddingDrink && v.renderedMode != AddingDrink {
		v.drinkSearch.SetText(s.Form.DrinkQuery)
		v.drinkTags.SetCSV(s.Form.Tags)
	}
	if s.Mode == Analyzing && v.renderedMode != Analyzing {
		v.targetMargin.SetText(s.AnalysisForm.TargetMargin)
	}
	if (s.Mode == Editing || s.Mode == Viewing || s.Mode == Creating || s.Mode == Tagging) && (v.renderedMode != s.Mode || v.renderedInstance != s.FormInstance || !reflect.DeepEqual(v.renderedForm, s.Form)) {
		v.populate(s.Form)
		v.renderedMode, v.renderedInstance, v.renderedForm = s.Mode, s.FormInstance, s.Form
	}
	busy := s.Loading || s.Submitting || s.Confirming
	detail := s.Selected != nil && (s.Mode == Viewing || s.Mode == Editing)
	v.addDrink.Hidden = !detail || !actionVisible(s, menusdomain.ControlAddDrink)
	v.analyze.Hidden = !detail
	v.publish.Hidden = !detail || !actionVisible(s, menusdomain.ControlPublish)
	v.draft.Hidden = !detail || !actionVisible(s, menusdomain.ControlDraft)
	v.tagAction.Hidden = !detail || !actionVisible(s, menusdomain.ControlTags)
	v.delete.Hidden = !detail || !actionVisible(s, menusdomain.ControlDelete)
	v.descriptionHelp.Hidden = s.Mode != Editing && s.Mode != Renaming
	listAccess := actionVisible(s, menusdomain.ControlList)
	setEnabled(v.refresh, !busy && listAccess)
	setEnabled(v.create, !busy)
	v.create.Hidden = !s.CanCreate
	setEnabled(v.filterExpression, !busy && listAccess)
	setEnabled(v.filterStatus, !busy && listAccess)
	setEnabled(v.applyFilter, !busy && listAccess)
	setEnabled(v.rename, !busy)
	setEnabled(v.tagAction, !busy)
	setEnabled(v.delete, !busy)
	v.refresh.Hidden = !listAccess
	v.empty.Hidden = !listAccess || s.Loading || len(s.Items) > 0
	v.list.Hidden = !listAccess || !v.empty.Hidden
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
	v.tags.SetEnabled(!busy && s.Mode != Viewing)
	v.drinkTags.SetEnabled(!busy)
	v.drinkChoices.RemoveAll()
	for _, option := range s.Drinks {
		b := ui.NewButton(controlDrinkChoicePrefix+option.ID.String(), option.Name, func() {
			f := v.p.State().Form
			f.Tags, f.ReplaceTags = v.drinkTags.CSV(), true
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
	return ui.ReadonlyEntry(value)
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
func actionVisible(s State, id actions.ID) bool { return s.Actions[id].Visible }
func actionEnabled(s State, id actions.ID) bool {
	return s.Actions[id].Visible && s.Actions[id].Enabled
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
