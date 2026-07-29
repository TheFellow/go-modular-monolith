package fyne

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
	ui "github.com/TheFellow/go-modular-monolith/pkg/fyne"
)

const (
	ControlRefresh          = "menus.refresh"
	ControlCreate           = "menus.create"
	ControlRename           = "menus.rename"
	ControlDelete           = "menus.delete"
	ControlPublish          = "menus.publish"
	ControlDraft            = "menus.draft"
	ControlTags             = "menus.tags"
	ControlAddDrink         = "menus.drink.add"
	ControlAnalyze          = "menus.analyze"
	ControlTargetMargin     = "menus.analysis.target-margin"
	ControlRunAnalysis      = "menus.analysis.run"
	ControlApplyFilter      = "menus.filter.apply"
	ControlFilterStatus     = "menus.filter.status"
	ControlFilterExpression = "menus.filter.expression"
	ControlPrevious         = "menus.previous"
	ControlNext             = "menus.next"
	ControlName             = "menus.form.name"
	ControlDescription      = "menus.form.description"
	ControlTagValues        = "menus.form.tags"
	ControlDrinkSearch      = "menus.drink.search"
	ControlSave             = "menus.form.save"
	ControlCancel           = "menus.form.cancel"
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
	p                                                                 *Presenter
	root, browse, formPanel, drinkPanel, analysisPanel                *framework.Container
	list                                                              *widget.List
	detail, status, formStatus, drinkStatus, analysisStatus           *widget.Label
	descriptionHelp                                                   *widget.Label
	detailBox                                                         *framework.Container
	removeActions                                                     *framework.Container
	filterStatus, filterLimit                                         *semanticSelect
	filterExpression, name, description, tags, drinkSearch, drinkTags *ui.SemanticEntry
	targetMargin                                                      *ui.SemanticEntry
	save, cancel, drinkCancel                                         *ui.SemanticButton
	refresh, create, rename, delete, publish, draft, analyze          *ui.SemanticButton
	tagAction, addDrink, applyFilter, drinkSearchAction               *ui.SemanticButton
	runAnalysis, analysisCancel                                       *ui.SemanticButton
	previous, next                                                    *ui.SemanticButton
	drinkChoices                                                      *framework.Container
	renderedMode                                                      Mode
	renderedForm                                                      Form
	formRendered                                                      bool
}

var _ ui.View = (*View)(nil)
var _ ui.Activated = (*View)(nil)

func NewView(p *Presenter) *View {
	v := &View{p: p}
	v.filterStatus = newSelect(ControlFilterStatus, []string{"", string(models.MenuStatusDraft), string(models.MenuStatusPublished), string(models.MenuStatusArchived)})
	v.filterExpression = ui.NewEntry(ControlFilterExpression)
	v.filterExpression.SetPlaceHolder("Expression filter")
	v.filterLimit = newSelect("menus.filter.limit", []string{"25", "50", "100"})
	v.filterLimit.SetSelected(strconv.Itoa(p.State().Filter.Limit))
	v.applyFilter = ui.NewButton(ControlApplyFilter, "Apply", func() {
		limit, _ := strconv.Atoi(v.filterLimit.Selected)
		if p.SetFilter(Filter{Status: models.MenuStatus(v.filterStatus.Selected), Expression: v.filterExpression.Text, Limit: limit}) {
			p.Refresh()
		}
	})
	filters := container.NewVBox(widget.NewLabelWithStyle("Filters", framework.TextAlignLeading, framework.TextStyle{Bold: true}), field("Status", v.filterStatus), field("Expression", v.filterExpression), field("Page size", v.filterLimit), v.applyFilter)
	v.list = widget.NewList(func() int { return len(p.State().Items) }, func() framework.CanvasObject { return widget.NewButton("", nil) }, func(i widget.ListItemID, object framework.CanvasObject) {
		menu := p.State().Items[i]
		button := object.(*widget.Button)
		button.SetText(fmt.Sprintf("%s  ·  %s", menu.Name, menu.Status))
		button.OnTapped = func() { p.Select(i) }
	})
	v.refresh = ui.NewButton(ControlRefresh, "Refresh", p.Refresh)
	v.create = ui.NewButton(ControlCreate, "New", p.StartCreate)
	v.rename = ui.NewButton(ControlRename, "Rename", p.StartRename)
	v.tagAction = ui.NewButton(ControlTags, "Tags", p.StartTags)
	v.addDrink = ui.NewButton(ControlAddDrink, "Add drink", p.StartAddDrink)
	v.publish = ui.NewButton(ControlPublish, "Publish", p.Publish)
	v.draft = ui.NewButton(ControlDraft, "Draft", p.ReturnToDraft)
	v.delete = ui.NewButton(ControlDelete, "Delete", p.Delete)
	v.analyze = ui.NewButton(ControlAnalyze, "Analyze", p.StartAnalysis)
	v.previous = ui.NewButton(ControlPrevious, "Previous", p.PreviousPage)
	v.next = ui.NewButton(ControlNext, "Next", p.NextPage)
	commands := container.NewGridWithColumns(11, v.refresh, v.create, v.rename, v.tagAction, v.addDrink, v.analyze, v.publish, v.draft, v.delete, v.previous, v.next)
	v.detail = widget.NewLabel("")
	v.detail.Wrapping = framework.TextWrapWord
	v.detailBox = container.NewVBox(v.detail)
	v.removeActions = container.NewHBox()
	v.status = widget.NewLabel("")
	v.browse = container.NewBorder(container.NewVBox(filters, commands), container.NewVBox(v.status, v.removeActions), nil, nil, ui.ListDetail(v.list, container.NewVScroll(v.detailBox), .35))
	v.name = ui.NewEntry(ControlName)
	v.description = ui.NewEntry(ControlDescription)
	v.description.MultiLine = true
	v.tags = ui.NewEntry(ControlTagValues)
	v.formStatus = widget.NewLabel("")
	v.save = ui.NewButton(ControlSave, "Save", func() { v.readForm(); p.Save() })
	v.cancel = ui.NewButton(ControlCancel, "Cancel", p.Cancel)
	v.descriptionHelp = widget.NewLabel("Leave description blank to keep the existing description.")
	v.formPanel = container.NewBorder(widget.NewLabelWithStyle("Menu", framework.TextAlignLeading, framework.TextStyle{Bold: true}), container.NewVBox(v.formStatus, container.NewHBox(layout.NewSpacer(), v.cancel, v.save)), nil, nil, container.NewVScroll(container.NewVBox(field("Name", v.name), field("Description", v.description), v.descriptionHelp, field("Complete tag set (CSV)", v.tags))))
	v.drinkSearch = ui.NewEntry(ControlDrinkSearch)
	v.drinkTags = ui.NewEntry(ControlTagValues + ".add-drink")
	v.drinkSearch.SetPlaceHolder("Search active drinks")
	v.drinkSearchAction = ui.NewButton(ControlDrinkSearch+".apply", "Search", func() { p.SearchDrinks(v.drinkSearch.Text) })
	v.drinkChoices = container.NewVBox()
	v.drinkStatus = widget.NewLabel("")
	v.drinkCancel = ui.NewButton(ControlCancel+".drink", "Cancel", p.Cancel)
	v.drinkPanel = container.NewBorder(container.NewVBox(widget.NewLabelWithStyle("Add drink", framework.TextAlignLeading, framework.TextStyle{Bold: true}), container.NewBorder(nil, nil, nil, v.drinkSearchAction, v.drinkSearch), field("Tags (complete set)", v.drinkTags)), container.NewVBox(v.drinkStatus, container.NewHBox(layout.NewSpacer(), v.drinkCancel)), nil, nil, container.NewVScroll(v.drinkChoices))
	v.targetMargin = ui.NewEntry(ControlTargetMargin)
	v.runAnalysis = ui.NewButton(ControlRunAnalysis, "Analyze", func() { p.SetAnalysisForm(AnalysisForm{TargetMargin: v.targetMargin.Text}); p.Analyze() })
	v.analysisCancel = ui.NewButton(ControlCancel+".analysis", "Close", p.Cancel)
	v.analysisStatus = widget.NewLabel("")
	v.analysisStatus.Wrapping = framework.TextWrapWord
	v.analysisPanel = container.NewBorder(container.NewVBox(widget.NewLabelWithStyle("Menu cost and availability analysis", framework.TextAlignLeading, framework.TextStyle{Bold: true}), field("Target margin (0–1)", v.targetMargin), v.runAnalysis), container.NewHBox(layout.NewSpacer(), v.analysisCancel), nil, nil, container.NewVScroll(v.analysisStatus))
	v.root = container.NewStack(v.browse, v.formPanel, v.drinkPanel, v.analysisPanel)
	p.Observe(v.render)
	return v
}
func (v *View) Title() string                   { return "Menus" }
func (v *View) Content() framework.CanvasObject { return v.root }
func (v *View) Activate()                       { v.p.Refresh() }
func (v *View) ExecuteCommand(command ui.Command) bool {
	state := v.p.State()
	switch command {
	case ui.CommandRefresh:
		return state.Mode == Browsing && ui.Trigger(v.refresh)
	case ui.CommandNew:
		return state.Mode == Browsing && ui.Trigger(v.create)
	case ui.CommandSave:
		if state.Mode == Analyzing {
			return ui.Trigger(v.runAnalysis)
		}
		return (state.Mode == Creating || state.Mode == Renaming || state.Mode == Tagging) && ui.Trigger(v.save)
	case ui.CommandCancel:
		switch state.Mode {
		case AddingDrink:
			return ui.Trigger(v.drinkCancel)
		case Analyzing:
			return ui.Trigger(v.analysisCancel)
		}
		return state.Mode != Browsing && ui.Trigger(v.cancel)
	default:
		return false
	}
}
func (v *View) readForm() {
	mode := v.p.State().Mode
	if mode == Tagging {
		v.p.SetForm(Form{Tags: v.tags.Text})
		return
	}
	v.p.SetForm(Form{Name: v.name.Text, Description: v.description.Text, Tags: v.tags.Text, ReplaceTags: true})
}
func (v *View) render(state State) {
	v.browse.Hidden = state.Mode != Browsing
	v.formPanel.Hidden = state.Mode != Creating && state.Mode != Renaming && state.Mode != Tagging
	v.drinkPanel.Hidden = state.Mode != AddingDrink
	v.analysisPanel.Hidden = state.Mode != Analyzing
	v.descriptionHelp.Hidden = state.Mode != Renaming
	changed := !v.formRendered || v.renderedMode != state.Mode || !reflect.DeepEqual(v.renderedForm, state.Form)
	if changed {
		switch state.Mode {
		case Creating, Renaming:
			v.name.Show()
			v.description.Show()
			v.tags.Show()
			v.name.SetText(state.Form.Name)
			v.description.SetText(state.Form.Description)
			v.tags.SetText(state.Form.Tags)
		case Tagging:
			v.name.Hide()
			v.description.Hide()
			v.tags.Show()
			v.tags.SetText(state.Form.Tags)
		case AddingDrink:
			if v.renderedMode != AddingDrink {
				v.drinkSearch.SetText(state.Form.DrinkQuery)
				v.drinkTags.SetText(state.Form.Tags)
			}
		case Analyzing:
			if v.renderedMode != Analyzing {
				v.targetMargin.SetText(state.AnalysisForm.TargetMargin)
			}
		}
		v.renderedMode, v.renderedForm, v.formRendered = state.Mode, state.Form, true
	}
	busy := state.Submitting || state.Loading || state.Confirming
	selected := state.Selected
	draft := selected != nil && selected.Status == models.MenuStatusDraft
	published := selected != nil && selected.Status == models.MenuStatusPublished
	setEnabled(v.refresh, !busy)
	setEnabled(v.create, !busy)
	setEnabled(v.filterStatus, !busy)
	setEnabled(v.filterExpression, !busy)
	setEnabled(v.applyFilter, !busy)
	setEnabled(v.filterLimit, !busy)
	setEnabled(v.previous, !busy && len(state.History) > 0)
	setEnabled(v.next, !busy && state.Next != "")
	setEnabled(v.rename, !busy && draft)
	setEnabled(v.tagAction, !busy && selected != nil)
	setEnabled(v.addDrink, !busy && draft)
	setEnabled(v.publish, !busy && draft && len(selected.Items) > 0)
	setEnabled(v.draft, !busy && published)
	setEnabled(v.delete, !busy && draft)
	setEnabled(v.analyze, !busy && selected != nil)
	setEnabled(v.save, !busy)
	setEnabled(v.cancel, !busy)
	setEnabled(v.name, !busy)
	setEnabled(v.description, !busy)
	setEnabled(v.tags, !busy)
	drinkInteraction := !state.Submitting && !state.Confirming
	setEnabled(v.drinkSearch, drinkInteraction)
	setEnabled(v.drinkSearchAction, drinkInteraction)
	setEnabled(v.drinkCancel, drinkInteraction)
	setEnabled(v.targetMargin, !state.Loading)
	setEnabled(v.runAnalysis, !state.Loading)
	setEnabled(v.analysisCancel, !state.Loading)
	message := ""
	if state.Loading {
		message = "Loading…"
	} else if state.Submitting {
		message = "Saving…"
	} else if state.Confirming {
		message = "Awaiting confirmation…"
	} else if state.Err != nil {
		message = "Error: " + state.Err.Error()
	}
	v.formStatus.SetText(message)
	v.drinkStatus.SetText(message)
	if state.Analysis != nil {
		v.analysisStatus.SetText(analysisText(*state.Analysis))
	} else {
		v.analysisStatus.SetText(message)
	}
	v.list.Refresh()
	v.detailBox.RemoveAll()
	v.detailBox.Add(v.detail)
	v.removeActions.RemoveAll()
	if state.Selected == nil {
		v.detail.SetText("Select a menu")
	} else {
		v.detail.SetText(detailText(state.Selected, v.p.DrinkName))
		if state.Selected.Status == models.MenuStatusDraft {
			items := append([]models.MenuItem(nil), state.Selected.Items...)
			sort.SliceStable(items, func(i, j int) bool { return items[i].SortOrder < items[j].SortOrder })
			for _, item := range items {
				item := item
				button := ui.NewButton("menus.drink.remove."+item.DrinkID.String(), "Remove "+v.p.DrinkName(item.DrinkID), func() { v.p.RemoveDrink(item.DrinkID) })
				if busy {
					button.Disable()
				}
				v.removeActions.Add(button)
			}
		}
	}
	v.detailBox.Refresh()
	v.removeActions.Refresh()
	if state.Loading {
		v.status.SetText("Loading…")
	} else if state.Err != nil {
		v.status.SetText("Error: " + state.Err.Error())
	} else {
		v.status.SetText(fmt.Sprintf("%d menus", len(state.Items)))
	}
	v.drinkChoices.RemoveAll()
	for _, option := range state.Drinks {
		option := option
		button := ui.NewButton("menus.drink.choice."+option.ID.String(), fmt.Sprintf("%s  ·  %s", option.Name, option.ID.String()), func() {
			form := v.p.State().Form
			form.Tags, form.ReplaceTags = v.drinkTags.Text, true
			v.p.SetForm(form)
			v.p.AddDrink(option.ID)
		})
		if busy {
			button.Disable()
		}
		v.drinkChoices.Add(button)
	}
	if len(state.Drinks) == 0 && state.Mode == AddingDrink && !state.Loading {
		v.drinkChoices.Add(widget.NewLabel("No matching drinks"))
	}
	v.root.Refresh()
}
func analysisText(analysis queries.MenuAnalytics) string {
	lines := []string{fmt.Sprintf("Available: %d/%d", analysis.AvailableCount, analysis.TotalCount)}
	if analysis.AverageMargin != nil {
		lines = append(lines, fmt.Sprintf("Average margin: %.0f%%", *analysis.AverageMargin*100))
	}
	for _, item := range analysis.Items {
		cost := "n/a"
		if item.Cost != nil && !item.CostUnknown {
			cost = item.Cost.String()
		}
		price := "n/a"
		if item.MenuPrice != nil {
			price = item.MenuPrice.String()
		} else if item.SuggestedPrice != nil {
			price = "suggested " + item.SuggestedPrice.String()
		}
		margin := "n/a"
		if item.Margin != nil {
			margin = fmt.Sprintf("%.0f%%", *item.Margin*100)
		}
		status := strings.ToUpper(string(item.Availability))
		if len(item.Substitutions) > 0 {
			sub := item.Substitutions[0]
			status += fmt.Sprintf(" (sub: %s for %s)", sub.Substitute.String(), sub.Original.String())
		}
		lines = append(lines, fmt.Sprintf("\n%s\nID: %s\nCost: %s\nPrice: %s\nMargin: %s\nStatus: %s", item.Name, item.DrinkID.String(), cost, price, margin, status))
	}
	return strings.Join(lines, "\n")
}
func setEnabled(object interface {
	Enable()
	Disable()
}, enabled bool) {
	if enabled {
		object.Enable()
	} else {
		object.Disable()
	}
}
func field(label string, object framework.CanvasObject) framework.CanvasObject {
	return container.NewBorder(nil, nil, widget.NewLabel(label), nil, object)
}
func detailText(menu *models.Menu, drinkName func(entity.DrinkID) string) string {
	tags := menu.Tags.Canonical().String()
	if tags == "" {
		tags = "None"
	}
	parts := []string{menu.Name, "", "ID: " + menu.ID.String(), "Created: " + menu.CreatedAt.Format(time.RFC3339), "Status: " + string(menu.Status), "Tags: " + tags}
	if publishedAt, ok := menu.PublishedAt.Unwrap(); ok {
		parts = append(parts, "Published: "+publishedAt.Format(time.RFC3339))
	}
	if menu.Description != "" {
		parts = append(parts, "", "Description", menu.Description)
	}
	parts = append(parts, "", fmt.Sprintf("Menu items (%d)", len(menu.Items)))
	items := append([]models.MenuItem(nil), menu.Items...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].SortOrder < items[j].SortOrder })
	if len(items) == 0 {
		parts = append(parts, "No drinks added")
	}
	for _, item := range items {
		fields := []string{drinkName(item.DrinkID), "Drink ID: " + item.DrinkID.String(), fmt.Sprintf("Sort order: %d", item.SortOrder), string(item.Availability)}
		if price, ok := item.Price.Unwrap(); ok {
			fields = append(fields, price.String())
		} else {
			fields = append(fields, "N/A")
		}
		if item.Featured {
			fields = append(fields, "featured")
		}
		parts = append(parts, "• "+strings.Join(fields, " | "))
	}
	return strings.Join(parts, "\n")
}
