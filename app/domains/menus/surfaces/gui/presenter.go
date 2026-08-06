// Package gui implements the retained-mode desktop surface for menus.
package gui

import (
	"context"
	"fmt"
	"maps"
	"math"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	drinks "github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	drinkmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	menus "github.com/TheFellow/go-modular-monolith/app/domains/menus"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/queries"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	"github.com/TheFellow/go-modular-monolith/pkg/set"
	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

type Mode uint8

const (
	Browsing Mode = iota
	Viewing
	Editing
	Creating
	Renaming
	Tagging
	AddingDrink
	Analyzing
)

type Filter struct {
	Status     models.MenuStatus
	Expression string
	Limit      int
}

type Form struct {
	Name, Description, Tags, DrinkQuery string
	ReplaceTags                         bool
}
type AnalysisForm struct{ TargetMargin string }
type DrinkOption struct {
	ID   entity.DrinkID
	Name string
}
type State struct {
	Mode                            Mode
	Loading, Submitting, Confirming bool
	Items                           []*models.Menu
	Selected                        *models.Menu
	Filter                          Filter
	Cursor, Next                    paging.Cursor
	History                         []paging.Cursor
	Form                            Form
	Drinks                          []DrinkOption
	AnalysisForm                    AnalysisForm
	Analysis                        *queries.MenuAnalytics
	Readiness                       *models.ReadinessReport
	Err                             error
	Dirty                           bool
	CanUpdate, CanDelete, CanTag    bool
	CanAddDrink, CanRemoveDrink     bool
	CanPublish, CanDraft            bool
	CanList                         bool
	CanCreate                       bool
	Actions                         map[actions.ID]actions.State
	FormInstance                    uint64
}
type Dependencies struct {
	Executor   ui.Executor
	Dispatcher ui.Dispatcher
	Dialogs    ui.Dialogs
	Projector  *menus.ActionProjector
}
type catalog struct {
	menus []*models.Menu
	names map[entity.DrinkID]string
	next  paging.Cursor
}

type Presenter struct {
	app           *app.Session
	dialogs       ui.Dialogs
	load          *ui.LatestRequest[catalog]
	choices       *ui.LatestRequest[[]DrinkOption]
	analysis      *ui.LatestRequest[queries.MenuAnalytics]
	submit        *ui.Submission
	state         State
	names         map[entity.DrinkID]string
	changed       func(State)
	confirming    bool
	projector     menus.ActionProjector
	projectionErr error
}

func NewPresenter(session *app.Session, d Dependencies) *Presenter {
	projector := menus.NewActionProjector()
	if d.Projector != nil {
		projector = *d.Projector
	}
	p := &Presenter{app: session, dialogs: d.Dialogs, names: make(map[entity.DrinkID]string), projector: projector, state: State{Filter: Filter{Limit: ui.PageLimit}}}
	if err := p.permissionsFor(nil); err != nil {
		p.state.Err = ui.PresentError(err)
		ui.ShowPresentation(p.dialogs, err)
	}
	p.load = ui.NewLatestRequest[catalog](d.Executor, d.Dispatcher)
	p.choices = ui.NewLatestRequest[[]DrinkOption](d.Executor, d.Dispatcher)
	p.analysis = ui.NewLatestRequest[queries.MenuAnalytics](d.Executor, d.Dispatcher)
	p.submit = ui.NewSubmission(d.Executor, d.Dispatcher)
	return p
}
func (p *Presenter) Observe(fn func(State)) { p.changed = fn; p.publish() }
func (p *Presenter) State() State           { return cloneState(p.state) }
func (p *Presenter) SetFilter(filter Filter) bool {
	if p.state.Submitting || p.state.Confirming || !p.actionEnabled(menus.ControlList) {
		return false
	}
	if filter.Limit < 0 {
		p.fail(errors.Invalidf("page size must be greater than zero"))
		return false
	}
	if filter.Limit == 0 {
		filter.Limit = ui.PageLimit
	}
	filter.Expression = strings.TrimSpace(filter.Expression)
	p.state.Filter, p.state.Cursor, p.state.Next, p.state.History, p.state.Err = filter, "", "", nil, nil
	p.publish()
	return true
}
func (p *Presenter) Refresh() {
	if !p.actionEnabled(menus.ControlList) {
		return
	}
	p.state.Cursor, p.state.Next, p.state.History = "", "", nil
	p.loadPage(false)
}

func (p *Presenter) loadPage(appendPage bool) {
	if !p.actionEnabled(menus.ControlList) {
		return
	}
	f := p.state.Filter
	cursor := p.state.Cursor
	p.load.LoadContext(p.app.Context(), func(ctx context.Context) (catalog, error) {
		op := p.app.ContextFrom(ctx)
		page, err := p.app.Menus.List(op, menus.ListRequest{Status: f.Status, Filter: f.Expression, Cursor: cursor, Limit: f.Limit})
		if err != nil {
			return catalog{}, err
		}
		names := make(map[entity.DrinkID]string)
		for _, menu := range page.Items {
			if menu == nil {
				return catalog{}, errors.Internalf("menu missing")
			}
			for _, item := range menu.Items {
				if _, ok := names[item.DrinkID]; ok {
					continue
				}
				name, ok := item.DisplayName.Unwrap()
				if strings.TrimSpace(name) == "" || !ok {
					drink, getErr := p.app.Drinks.Get(op, item.DrinkID)
					if getErr != nil {
						return catalog{}, getErr
					}
					if drink == nil {
						return catalog{}, errors.Internalf("drink %s missing", item.DrinkID.String())
					}
					name = drink.Name
				}
				names[item.DrinkID] = name
			}
		}
		return catalog{menus: page.Items, names: names, next: page.Next}, nil
	}, func(r ui.LoadState[catalog]) {
		p.state.Loading = r.Status == ui.Loading
		if r.Status == ui.Failed {
			p.state.Err = ui.PresentError(r.Err)
			ui.ShowPresentation(p.dialogs, r.Err)
		}
		if r.Status == ui.Loaded {
			p.state.Err = nil
			if appendPage {
				p.state.Items = append(p.state.Items, cloneMenus(r.Value.menus)...)
			} else {
				p.state.Items = cloneMenus(r.Value.menus)
			}
			p.state.Next = r.Value.next
			if appendPage {
				maps.Copy(p.names, r.Value.names)
			} else {
				p.names = cloneNames(r.Value.names)
			}
			p.reselect()
		}
		p.publish()
	})
}
func (p *Presenter) NextPage() {
	if p.state.Next == "" || p.state.Loading || !p.actionEnabled(menus.ControlList) {
		return
	}
	p.state.History = append(p.state.History, p.state.Cursor)
	p.state.Cursor = p.state.Next
	p.loadPage(true)
}
func (p *Presenter) PreviousPage() {
	if len(p.state.History) == 0 || p.state.Loading || !p.actionEnabled(menus.ControlList) {
		return
	}
	last := len(p.state.History) - 1
	p.state.Cursor = p.state.History[last]
	p.state.History = p.state.History[:last]
	p.loadPage(false)
}
func (p *Presenter) Select(index int) {
	if p.state.Loading || p.state.Submitting || p.state.Confirming || !p.actionEnabled(menus.ControlList) {
		return
	}
	if index < 0 || index >= len(p.state.Items) {
		p.state.Selected = nil
		p.state.Mode = Browsing
	} else {
		p.state.Selected = cloneMenu(p.state.Items[index])
		p.state.FormInstance++
		p.state.Form = formFromMenu(p.state.Selected)
		p.permissions()
		p.loadReadiness()
		if p.state.CanUpdate && p.state.Selected.Status == models.MenuStatusDraft {
			p.state.Mode = Editing
		} else {
			p.state.Mode = Viewing
		}
		p.state.Dirty, p.state.Err = false, nil
	}
	p.publish()
}

func (p *Presenter) ListActions(index int) (map[actions.ID]actions.State, error) {
	if !p.actionEnabled(menus.ControlList) || index < 0 || index >= len(p.state.Items) || p.state.Items[index] == nil {
		return nil, nil
	}
	states, err := p.projector.Project(p.app.Context(), p.app.Context().Principal(), p.state.Items[index])
	if err != nil {
		p.projectionErr = err
		return nil, err
	}
	p.clearProjectionError()
	return indexActions(states), err
}

func (p *Presenter) Back()      { p.leaveDetail(false) }
func (p *Presenter) ResetList() { p.leaveDetail(true) }
func (p *Presenter) leaveDetail(reset bool) {
	if p.submit.Active() {
		return
	}
	proceed := func() {
		if reset {
			p.state.Filter = Filter{Limit: ui.PageLimit}
			p.state.Cursor, p.state.Next, p.state.History = "", "", nil
		}
		p.state.Mode, p.state.Dirty, p.state.Err = Browsing, false, nil
		p.publish()
		if reset {
			p.Refresh()
		}
	}
	if p.state.Dirty {
		if p.dialogs == nil {
			return
		}
		p.dialogs.Confirm("Discard changes?", "Discard unsaved menu changes?", func(ok bool) {
			if ok {
				proceed()
			}
		})
		return
	}
	proceed()
}
func (p *Presenter) StartCreate() {
	if p.state.Loading || p.state.Submitting || p.state.Confirming || !p.state.CanCreate {
		return
	}
	p.state.FormInstance++
	p.state.Mode, p.state.Form, p.state.Err, p.state.Dirty = Creating, Form{ReplaceTags: true}, nil, false
	p.publish()
}
func (p *Presenter) StartRename() {
	if p.state.Loading || p.state.Submitting || p.state.Confirming || p.state.Dirty || !p.actionEnabled(menus.ControlEdit) {
		return
	}
	p.state.FormInstance++
	p.state.Mode, p.state.Form, p.state.Err = Renaming, formFromMenu(p.state.Selected), nil
	p.publish()
}
func (p *Presenter) StartTags() {
	if p.state.Loading || p.state.Submitting || p.state.Confirming || p.state.Dirty || !p.actionEnabled(menus.ControlTags) {
		return
	}
	values, _ := tag.FormatCollection(p.state.Selected.Tags)
	p.state.Mode, p.state.Form, p.state.Err = Tagging, Form{Tags: values}, nil
	p.publish()
}
func (p *Presenter) StartAddDrink() {
	if p.state.Loading || p.state.Submitting || p.state.Confirming || p.state.Dirty || !p.actionEnabled(menus.ControlAddDrink) {
		return
	}
	p.state.Mode, p.state.Form, p.state.Drinks, p.state.Err = AddingDrink, Form{Tags: p.state.Selected.Tags.Canonical().String(), ReplaceTags: true}, nil, nil
	p.publish()
	p.SearchDrinks("")
}
func (p *Presenter) StartAnalysis() {
	if p.state.Loading || p.state.Submitting || p.state.Confirming || p.state.Selected == nil {
		return
	}
	p.state.Mode, p.state.AnalysisForm, p.state.Analysis, p.state.Err = Analyzing, AnalysisForm{TargetMargin: "0.70"}, nil, nil
	p.publish()
}
func (p *Presenter) SetAnalysisForm(form AnalysisForm) { p.state.AnalysisForm = form; p.publish() }
func (p *Presenter) Analyze() bool {
	if p.state.Mode != Analyzing || p.state.Selected == nil {
		p.fail(errors.Invalidf("menu analysis is not active"))
		return false
	}
	target, err := strconv.ParseFloat(strings.TrimSpace(p.state.AnalysisForm.TargetMargin), 64)
	if err != nil || math.IsNaN(target) || math.IsInf(target, 0) || target <= 0 || target >= 1 {
		p.fail(errors.Invalidf("target margin must be a number between 0 and 1"))
		return false
	}
	menu := *cloneMenu(p.state.Selected)
	p.analysis.LoadContext(p.app.Context(), func(ctx context.Context) (queries.MenuAnalytics, error) {
		return p.app.Menus.Analyze(p.app.ContextFrom(ctx), menu, target)
	}, func(r ui.LoadState[queries.MenuAnalytics]) {
		p.state.Loading = r.Status == ui.Loading
		if r.Status == ui.Failed {
			p.state.Err = ui.PresentError(r.Err)
			ui.ShowPresentation(p.dialogs, r.Err)
		}
		if r.Status == ui.Loaded {
			value := cloneAnalysis(r.Value)
			p.state.Analysis, p.state.Err = &value, nil
		}
		p.publish()
	})
	return true
}
func (p *Presenter) SearchDrinks(query string) {
	if p.state.Mode != AddingDrink {
		return
	}
	p.state.Form.DrinkQuery = query
	menuID := p.state.Selected.ID
	p.choices.LoadContext(p.app.Context(), func(ctx context.Context) ([]DrinkOption, error) {
		op := p.app.ContextFrom(ctx)
		all, err := paging.Collect(func(cursor paging.Cursor) (paging.Page[*drinkmodels.Drink], error) {
			return p.app.Drinks.List(op, drinks.ListRequest{Cursor: cursor})
		})
		if err != nil {
			return nil, err
		}
		menu, err := p.app.Menus.Get(op, menuID)
		if err != nil {
			return nil, err
		}
		var included set.Set[entity.DrinkID]
		for _, item := range menu.Items {
			included.Add(item.DrinkID)
		}
		needle := strings.ToLower(strings.TrimSpace(query))
		out := make([]DrinkOption, 0, len(all))
		for _, drink := range all {
			if drink != nil && !included.Contains(drink.ID) && (needle == "" || strings.Contains(strings.ToLower(drink.Name), needle)) {
				out = append(out, DrinkOption{ID: drink.ID, Name: drink.Name})
			}
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].Name == out[j].Name {
				return out[i].ID.String() < out[j].ID.String()
			}
			return out[i].Name < out[j].Name
		})
		return out, nil
	}, func(r ui.LoadState[[]DrinkOption]) {
		p.state.Loading = r.Status == ui.Loading
		if r.Status == ui.Failed {
			p.state.Err = ui.PresentError(r.Err)
			ui.ShowPresentation(p.dialogs, r.Err)
		}
		if r.Status == ui.Loaded {
			p.state.Drinks = append([]DrinkOption(nil), r.Value...)
			p.state.Err = nil
		}
		p.publish()
	})
}
func (p *Presenter) SetForm(form Form) {
	p.state.Form = form
	p.state.Dirty = p.state.Selected != nil && !reflect.DeepEqual(form, formFromMenu(p.state.Selected))
	p.publish()
}
func (p *Presenter) Cancel() {
	if p.submit.Active() {
		return
	}
	p.choices.Invalidate()
	p.analysis.Invalidate()
	if (p.state.Mode == Editing || p.state.Mode == Viewing) && p.state.Selected != nil {
		p.state.Form, p.state.Dirty, p.state.Err = formFromMenu(p.state.Selected), false, nil
		p.state.FormInstance++
		p.publish()
		return
	}
	p.state.Mode, p.state.Err, p.state.Dirty = Browsing, nil, false
	p.publish()
}
func (p *Presenter) Save() bool {
	mode, form, target := p.state.Mode, p.state.Form, cloneMenu(p.state.Selected)
	if mode == Creating && !p.actionEnabled(menus.ControlCreate) || mode == Renaming && !p.actionEnabled(menus.ControlEdit) || mode == Tagging && !p.actionEnabled(menus.ControlTags) {
		return false
	}
	switch mode {
	case Creating, Renaming, Editing:
		if mode == Editing && (!p.state.CanUpdate || !p.state.Dirty) {
			return false
		}
		var desired *tag.Tags
		if form.ReplaceTags {
			values, err := tag.ParseCollection(form.Tags)
			if err != nil {
				p.fail(err)
				return false
			}
			desired = &values
		}
		name := strings.TrimSpace(form.Name)
		if name == "" {
			p.fail(errors.Invalidf("name is required"))
			return false
		}
		if len([]rune(name)) > 100 {
			p.fail(errors.Invalidf("name must be at most 100 characters"))
			return false
		}
		description := strings.TrimSpace(form.Description)
		if len([]rune(description)) > 500 {
			p.fail(errors.Invalidf("description must be at most 500 characters"))
			return false
		}
		return p.mutate(func() error {
			if mode == Creating {
				_, err := app.RunTaggedMutation(p.app.App, p.app.Context(), desired, func(ctx *middleware.Context) (*models.Menu, error) {
					return p.app.Menus.Create(ctx, &models.Menu{Name: name, Description: description})
				})
				return err
			}
			if target == nil {
				return errors.Invalidf("menu not selected")
			}
			_, err := app.RunTaggedMutation(p.app.App, p.app.Context(), desired, func(ctx *middleware.Context) (*models.Menu, error) {
				return p.app.Menus.Update(ctx, &models.Menu{ID: target.ID, Name: name, Description: description})
			})
			return err
		})
	case Tagging:
		if target == nil {
			p.fail(errors.Invalidf("menu not selected"))
			return false
		}
		tags, err := tag.ParseCollection(form.Tags)
		if err != nil {
			p.fail(err)
			return false
		}
		return p.mutate(func() error { _, err := p.app.Tags.Replace(p.app.Context(), target.EntityUID(), tags); return err })
	case Browsing, Viewing, AddingDrink, Analyzing:
		p.fail(errors.Invalidf("no menu form is active"))
		return false
	}
	return false
}
func (p *Presenter) AddDrink(id entity.DrinkID) bool {
	target := cloneMenu(p.state.Selected)
	if target == nil || !p.actionEnabled(menus.ControlAddDrink) {
		p.fail(errors.Invalidf("draft menu is required"))
		return false
	}
	desired, err := taggedChoice(ui.ReplaceTags, p.state.Form.Tags)
	if !p.state.Form.ReplaceTags {
		desired, err = nil, nil
	}
	if err != nil {
		p.fail(err)
		return false
	}
	for _, option := range p.state.Drinks {
		if option.ID == id {
			p.names[id] = option.Name
			break
		}
	}
	return p.mutate(func() error {
		_, err := app.RunTaggedMutation(p.app.App, p.app.Context(), desired, func(ctx *middleware.Context) (*models.Menu, error) {
			return p.app.Menus.AddDrink(ctx, &models.MenuPatch{MenuID: target.ID, DrinkID: id})
		})
		return err
	})
}
func (p *Presenter) RemoveDrink(id entity.DrinkID) {
	target := cloneMenu(p.state.Selected)
	if target == nil || !p.actionEnabled(menus.ControlRemoveDrink) || p.state.Dirty || p.dialogs == nil || p.confirming || p.submit.Active() {
		return
	}
	p.confirming = true
	p.state.Confirming = true
	p.publish()
	ui.ConfirmTagged(p.dialogs, "Remove drink", fmt.Sprintf("Remove %q from %q?", p.DrinkName(id), target.Name), target.Tags.Canonical().String(), func(ok bool, mode ui.TagMutationMode, values string) {
		p.confirming = false
		p.state.Confirming = false
		p.publish()
		if ok {
			desired, err := taggedChoice(mode, values)
			if err != nil {
				p.fail(err)
				return
			}
			p.mutate(func() error {
				_, err := app.RunTaggedMutation(p.app.App, p.app.Context(), desired, func(ctx *middleware.Context) (*models.Menu, error) {
					return p.app.Menus.RemoveDrink(ctx, &models.MenuPatch{MenuID: target.ID, DrinkID: id})
				})
				return err
			})
		}
	})
}
func (p *Presenter) Delete() {
	target := cloneMenu(p.state.Selected)
	if target == nil || !p.actionEnabled(menus.ControlDelete) || p.state.Dirty || p.dialogs == nil || p.confirming || p.submit.Active() {
		return
	}
	p.confirming = true
	p.state.Confirming = true
	p.publish()
	message := fmt.Sprintf("Delete draft menu %q?", target.Name)
	if len(target.Items) > 0 {
		message += fmt.Sprintf("\n\nThis also removes %d menu item(s).", len(target.Items))
	}
	p.dialogs.Confirm("Delete menu", message, func(ok bool) {
		p.confirming = false
		p.state.Confirming = false
		p.publish()
		if ok {
			p.mutate(func() error { _, err := p.app.Menus.Delete(p.app.Context(), target.ID); return err })
		}
	})
}
func (p *Presenter) Publish() {
	target := cloneMenu(p.state.Selected)
	if target == nil || !p.actionEnabled(menus.ControlPublish) || p.state.Dirty || p.dialogs == nil || p.confirming || p.submit.Active() {
		return
	}
	p.confirming = true
	p.state.Confirming = true
	p.publish()
	ui.ConfirmTagged(p.dialogs, "Publish menu", fmt.Sprintf("Publish %q with %d item(s)?", target.Name, len(target.Items)), target.Tags.Canonical().String(), func(ok bool, mode ui.TagMutationMode, values string) {
		p.confirming = false
		p.state.Confirming = false
		p.publish()
		if ok {
			desired, err := taggedChoice(mode, values)
			if err != nil {
				p.fail(err)
				return
			}
			p.mutate(func() error {
				_, err := app.RunTaggedMutation(p.app.App, p.app.Context(), desired, func(ctx *middleware.Context) (*models.Menu, error) { return p.app.Menus.Publish(ctx, target) })
				return err
			})
		}
	})
}
func (p *Presenter) ReturnToDraft() {
	target := cloneMenu(p.state.Selected)
	if target == nil || !p.actionEnabled(menus.ControlDraft) || p.state.Dirty || p.dialogs == nil || p.confirming || p.submit.Active() {
		return
	}
	p.confirming = true
	p.state.Confirming = true
	p.publish()
	ui.ConfirmTagged(p.dialogs, "Return to draft", fmt.Sprintf("Return published menu %q to draft?", target.Name), target.Tags.Canonical().String(), func(ok bool, mode ui.TagMutationMode, values string) {
		p.confirming = false
		p.state.Confirming = false
		p.publish()
		if ok {
			desired, err := taggedChoice(mode, values)
			if err != nil {
				p.fail(err)
				return
			}
			p.mutate(func() error {
				_, err := app.RunTaggedMutation(p.app.App, p.app.Context(), desired, func(ctx *middleware.Context) (*models.Menu, error) { return p.app.Menus.Draft(ctx, target) })
				return err
			})
		}
	})
}

func taggedChoice(mode ui.TagMutationMode, values string) (*tag.Tags, error) {
	if mode == ui.PreserveTags {
		return nil, nil
	}
	if mode == ui.ClearTags {
		empty := tag.Tags{}
		return &empty, nil
	}
	parsed, err := tag.ParseCollection(values)
	return &parsed, err
}
func (p *Presenter) DrinkName(id entity.DrinkID) string {
	if name := strings.TrimSpace(p.names[id]); name != "" {
		return name
	}
	return id.String()
}
func (p *Presenter) mutate(work func() error) bool {
	if p.submit.Active() {
		return false
	}
	p.state.Submitting, p.state.Err = true, nil
	p.publish()
	accepted := p.submit.Submit(work, func(err error) {
		p.state.Submitting = false
		if err != nil {
			p.fail(err)
			return
		}
		if p.state.Selected != nil && p.state.Mode != Creating {
			updated, getErr := p.app.Menus.Get(p.app.Context(), p.state.Selected.ID)
			if getErr != nil {
				p.fail(getErr)
				return
			}
			p.state.Selected = cloneMenu(updated)
			p.state.Form = formFromMenu(updated)
			p.state.Dirty, p.state.Err = false, nil
			p.permissions()
			if p.state.Mode == Editing || p.state.Mode == Viewing {
				if p.state.CanUpdate && updated.Status == models.MenuStatusDraft {
					p.state.Mode = Editing
				} else {
					p.state.Mode = Viewing
				}
			} else {
				p.state.Mode = Viewing
			}
			for i, item := range p.state.Items {
				if item.ID == updated.ID {
					p.state.Items[i] = cloneMenu(updated)
				}
			}
			p.publish()
		} else {
			p.state.Mode, p.state.Err = Browsing, nil
			p.publish()
			p.Refresh()
		}
	})
	if !accepted {
		p.state.Submitting = false
		p.publish()
	}
	return accepted
}

func formFromMenu(menu *models.Menu) Form {
	if menu == nil {
		return Form{}
	}
	return Form{Name: menu.Name, Description: menu.Description, Tags: menu.Tags.Canonical().String(), ReplaceTags: true}
}
func (p *Presenter) permissions() {
	if err := p.permissionsFor(p.state.Selected); err != nil {
		p.state.Err = ui.PresentError(err)
		ui.ShowPresentation(p.dialogs, err)
	}
}
func (p *Presenter) loadReadiness() {
	p.state.Readiness = nil
	if p.state.Selected == nil {
		return
	}
	state := p.state.Actions[menus.ControlReadiness]
	if !state.Visible || !state.Enabled {
		return
	}
	report, err := p.app.Menus.Readiness(p.app.Context(), p.state.Selected.ID)
	if err != nil {
		p.state.Err = ui.PresentError(err)
		return
	}
	p.state.Readiness = &report
	if report.HasBlockers() {
		state := p.state.Actions[menus.ControlPublish]
		if state.Visible && state.Enabled {
			state.Enabled = false
			state.DisabledReason = "Resolve menu readiness blockers before publishing."
			p.state.Actions[menus.ControlPublish] = state
		}
	}
}
func (p *Presenter) permissionsFor(menu *models.Menu) error {
	p.state.Actions = nil
	p.state.CanList, p.state.CanCreate, p.state.CanUpdate, p.state.CanDelete, p.state.CanTag = false, false, false, false, false
	p.state.CanAddDrink, p.state.CanRemoveDrink, p.state.CanPublish, p.state.CanDraft = false, false, false, false
	states, err := p.projector.Project(p.app.Context(), p.app.Context().Principal(), menu)
	if err != nil {
		p.projectionErr = err
		return err
	}
	p.clearProjectionError()
	p.state.Actions = indexActions(states)
	state := func(id actions.ID) actions.State { return p.state.Actions[id] }
	p.state.CanList = state(menus.ControlList).Visible
	p.state.CanCreate = state(menus.ControlCreate).Visible
	p.state.CanUpdate = state(menus.ControlEdit).Visible
	p.state.CanDelete = state(menus.ControlDelete).Visible
	p.state.CanTag = state(menus.ControlTags).Visible
	p.state.CanAddDrink = state(menus.ControlAddDrink).Visible
	p.state.CanRemoveDrink = state(menus.ControlRemoveDrink).Visible
	p.state.CanPublish = state(menus.ControlPublish).Visible
	p.state.CanDraft = state(menus.ControlDraft).Visible
	return nil
}
func (p *Presenter) clearProjectionError() {
	if p.projectionErr != nil && errors.Is(p.state.Err, p.projectionErr) {
		p.state.Err = nil
	}
	p.projectionErr = nil
}
func (p *Presenter) actionEnabled(id actions.ID) bool {
	state, ok := p.state.Actions[id]
	return ok && state.Visible && state.Enabled
}
func indexActions(states []actions.State) map[actions.ID]actions.State {
	indexed := make(map[actions.ID]actions.State, len(states))
	for _, state := range states {
		indexed[state.ID] = state
	}
	return indexed
}
func (p *Presenter) fail(err error) {
	p.state.Err = ui.PresentError(err)
	ui.ShowPresentation(p.dialogs, err)
	p.publish()
}
func (p *Presenter) recordProjectionError(err error) {
	if err == nil || p.state.Err != nil && p.state.Err.Error() == ui.PresentError(err).Error() {
		return
	}
	p.state.Err = ui.PresentError(err)
	ui.ShowPresentation(p.dialogs, err)
}
func (p *Presenter) publish() {
	if p.changed != nil {
		p.changed(cloneState(p.state))
	}
}
func (p *Presenter) reselect() {
	if p.state.Selected == nil {
		if len(p.state.Items) > 0 {
			p.state.Selected = cloneMenu(p.state.Items[0])
			p.permissions()
			p.loadReadiness()
		}
		return
	}
	id := p.state.Selected.ID
	for _, item := range p.state.Items {
		if item.ID == id {
			p.state.Selected = cloneMenu(item)
			p.permissions()
			p.loadReadiness()
			return
		}
	}
	p.state.Selected = nil
	p.permissions()
	p.state.Readiness = nil
}
func cloneMenu(in *models.Menu) *models.Menu {
	if in == nil {
		return nil
	}
	out := *in
	out.Items = append([]models.MenuItem(nil), in.Items...)
	out.Tags = append(tag.Tags(nil), in.Tags...)
	return &out
}
func cloneMenus(in []*models.Menu) []*models.Menu {
	out := make([]*models.Menu, len(in))
	for i := range in {
		out[i] = cloneMenu(in[i])
	}
	return out
}
func cloneNames(in map[entity.DrinkID]string) map[entity.DrinkID]string {
	out := make(map[entity.DrinkID]string, len(in))
	maps.Copy(out, in)
	return out
}
func cloneState(in State) State {
	in.Items = cloneMenus(in.Items)
	in.History = append([]paging.Cursor(nil), in.History...)
	in.Selected = cloneMenu(in.Selected)
	in.Drinks = append([]DrinkOption(nil), in.Drinks...)
	in.Actions = maps.Clone(in.Actions)
	if in.Analysis != nil {
		value := cloneAnalysis(*in.Analysis)
		in.Analysis = &value
	}
	if in.Readiness != nil {
		value := *in.Readiness
		value.Findings = slices.Clone(in.Readiness.Findings)
		in.Readiness = &value
	}
	return in
}
func cloneAnalysis(in queries.MenuAnalytics) queries.MenuAnalytics {
	in.Menu = *cloneMenu(&in.Menu)
	in.Items = append([]queries.MenuItemAnalytics(nil), in.Items...)
	for i := range in.Items {
		in.Items[i].Substitutions = slices.Clone(in.Items[i].Substitutions)
		if in.Items[i].Cost != nil {
			value := *in.Items[i].Cost
			in.Items[i].Cost = &value
		}
		if in.Items[i].MenuPrice != nil {
			value := *in.Items[i].MenuPrice
			in.Items[i].MenuPrice = &value
		}
		if in.Items[i].Margin != nil {
			value := *in.Items[i].Margin
			in.Items[i].Margin = &value
		}
		if in.Items[i].SuggestedPrice != nil {
			value := *in.Items[i].SuggestedPrice
			in.Items[i].SuggestedPrice = &value
		}
	}
	if in.AverageMargin != nil {
		value := *in.AverageMargin
		in.AverageMargin = &value
	}
	return in
}
