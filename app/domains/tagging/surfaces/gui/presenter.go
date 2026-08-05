// Package gui provides the bespoke retained-mode desktop presentation for tagging.
package gui

import (
	"context"
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus"
	menusmodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
	cedar "github.com/cedar-policy/cedar-go"
)

type Operation uint8

const (
	Inspect Operation = iota
	Add
	Remove
	ShowExact
	ShowKey
	Summary
)

func (o Operation) valid() bool {
	switch o {
	case Inspect, Add, Remove, ShowExact, ShowKey, Summary:
		return true
	}
	return false
}

type Mode uint8

const (
	Browsing Mode = iota
	PickingType
	PickingEntity
	EnteringValue
	Loading
	Results
)

type EntityOption struct {
	UID          cedar.EntityUID
	Name, Detail string
	Resource     cedar.Entity
	Actions      map[actions.ID]actions.State
}

type Result struct {
	Target     cedar.EntityUID
	TargetName string
	Tags       tag.Tags
	Changed    bool
	Inspected  bool
	References []tagging.Reference
	Summaries  []tagging.Summary
}

type State struct {
	Mode              Mode
	Operation         Operation
	EntityType        cedar.EntityType
	Entities          []EntityOption
	Visible           []EntityOption
	Query             string
	Target            cedar.EntityUID
	TargetName        string
	Value             string
	Result            Result
	Catalog           []tagging.Summary
	VisibleSummaries  []tagging.Summary
	Submitting        bool
	Err               error
	SummarySort       int
	SummaryDescending bool
	Actions           map[actions.ID]actions.State
}

type Dependencies struct {
	Executor   ui.Executor
	Dispatcher ui.Dispatcher
	Dialogs    ui.Dialogs
}

type Presenter struct {
	app       *app.Session
	dialogs   ui.Dialogs
	load      *ui.LatestRequest[any]
	submit    *ui.Submission
	state     State
	changed   func(State)
	projector tagging.ActionProjector
}

func NewPresenter(session *app.Session, deps Dependencies, projectors ...tagging.ActionProjector) *Presenter {
	var projector tagging.ActionProjector
	if session != nil && session.Tags != nil {
		projector = session.Tags.NewActionProjector()
	}
	if len(projectors) > 0 {
		projector = projectors[0]
	}
	p := &Presenter{app: session, dialogs: deps.Dialogs, load: ui.NewLatestRequest[any](deps.Executor, deps.Dispatcher), submit: ui.NewSubmission(deps.Executor, deps.Dispatcher), projector: projector}
	if session != nil {
		if states, err := projector.ProjectDiscovery(session.Context(), session.Context().Principal()); err != nil {
			p.state.Err = ui.PresentError(err)
		} else {
			p.state.Actions = actionMap(states)
		}
	} else {
		p.state.Actions = map[actions.ID]actions.State{
			tagging.ControlShow: {ID: tagging.ControlShow, Visible: true, Enabled: true}, tagging.ControlSummary: {ID: tagging.ControlSummary, Visible: true, Enabled: true},
		}
	}
	return p
}

func (p *Presenter) Observe(fn func(State)) { p.changed = fn; p.publish() }
func (p *Presenter) State() State           { return cloneState(p.state) }

func (p *Presenter) Start(operation Operation) {
	if p.state.Submitting {
		return
	}
	p.load.Invalidate()
	discovery := maps.Clone(p.state.Actions)
	p.state = State{Operation: operation, Actions: discovery}
	if !operation.valid() {
		p.fail(errors.Invalidf("invalid tag operation"))
		p.publish()
		return
	}
	if id := discoveryControl(operation); id != "" && !actionEnabled(p.state.Actions, id) {
		return
	}
	switch operation {
	case Inspect, Add, Remove:
		p.state.Mode = PickingType
	case ShowExact, ShowKey:
		p.state.Mode = EnteringValue
	case Summary:
		p.runQuery(func(ctx *middleware.Context) (any, error) { return p.app.Tags.Summary(ctx) })
	}
	p.publish()
}

func (p *Presenter) SelectType(kind cedar.EntityType) {
	if p.state.Submitting || p.state.Mode != PickingType || !supportedType(kind) {
		return
	}
	p.state.EntityType, p.state.Mode, p.state.Err = kind, Loading, nil
	p.publish()
	p.load.LoadContext(p.app.Context(), func(ctx context.Context) (any, error) { return p.loadEntities(p.app.ContextFrom(ctx), kind) }, func(r ui.LoadState[any]) {
		if r.Status == ui.Loading {
			return
		}
		if r.Status == ui.Failed {
			p.state.Mode, p.state.Err = PickingType, ui.PresentError(r.Err)
			ui.ShowPresentation(p.dialogs, r.Err)
			p.publish()
			return
		}
		options, ok := r.Value.([]EntityOption)
		if !ok {
			err := errors.Internalf("unexpected entity catalog")
			p.state.Mode, p.state.Err = PickingType, ui.PresentError(err)
			ui.ShowPresentation(p.dialogs, err)
			p.publish()
			return
		}
		for i := range options {
			states, err := p.projector.ProjectTarget(p.app.Context(), p.app.Context().Principal(), options[i].Resource)
			if err != nil {
				p.state.Mode, p.state.Err = PickingType, ui.PresentError(err)
				ui.ShowPresentation(p.dialogs, err)
				p.publish()
				return
			}
			options[i].Actions = actionMap(states)
		}
		p.state.Entities, p.state.Query, p.state.Mode, p.state.Err = cloneEntities(options), "", PickingEntity, nil
		p.applyQuery()
		p.publish()
	})
}

func (p *Presenter) Search(query string) {
	if p.state.Submitting {
		return
	}
	p.state.Query = query
	if p.state.Mode == PickingEntity {
		p.applyQuery()
	} else if p.state.Mode == Results && p.state.Operation == Summary {
		p.applySummaryQuery()
	} else {
		return
	}
	p.publish()
}

// SortSummaries sorts the complete in-memory tag summary result. Unlike the
// cursor-paged domain lists, this is never a misleading current-page sort.
func (p *Presenter) SortSummaries(column int, direction ui.SortDirection) {
	if p.state.Mode != Results || p.state.Operation != Summary || column < 0 || column > 6 {
		return
	}
	p.state.SummarySort = column + 1
	p.state.SummaryDescending = direction == ui.SortDescending
	p.sortSummaries()
	p.publish()
}

// SelectSummary opens the active references for a tag while retaining the
// exact summary filter so Back can restore the discovery list.
func (p *Presenter) SelectSummary(index int) {
	if p.state.Submitting || p.state.Mode != Results || p.state.Operation != Summary || index < 0 || index >= len(p.state.VisibleSummaries) {
		return
	}
	if !actionEnabled(p.state.Actions, tagging.ControlShow) {
		return
	}
	value, err := tag.Parse(p.state.VisibleSummaries[index].Tag)
	if err != nil {
		p.fail(err)
		return
	}
	p.state.Operation, p.state.Value = ShowExact, value.String()
	p.runQuery(func(ctx *middleware.Context) (any, error) { return p.app.Tags.Show(ctx, value, true) })
}

// ResetList is used by main navigation and the breadcrumb. Unlike Back it
// deliberately discards the retained summary filter.
func (p *Presenter) ResetList() {
	if p.state.Submitting || p.app == nil {
		return
	}
	if !actionEnabled(p.state.Actions, tagging.ControlSummary) {
		return
	}
	p.load.Invalidate()
	p.state = State{Operation: Summary, Actions: discoveryActions(p.state.Actions)}
	p.runQuery(func(ctx *middleware.Context) (any, error) { return p.app.Tags.Summary(ctx) })
}

func (p *Presenter) SelectEntity(index int) {
	if p.state.Submitting || p.state.Mode != PickingEntity || index < 0 || index >= len(p.state.Visible) {
		return
	}
	selected := p.state.Visible[index]
	if !actionEnabled(selected.Actions, targetControl(p.state.Operation)) {
		return
	}
	p.state.Target, p.state.TargetName, p.state.Value, p.state.Err = selected.UID, selected.Name, "", nil
	maps.Copy(p.state.Actions, selected.Actions)
	if p.state.Operation == Inspect {
		target := selected.UID
		p.runQuery(func(ctx *middleware.Context) (any, error) { return p.app.Tags.List(ctx, target) })
		return
	}
	p.state.Mode = EnteringValue
	p.publish()
}

func (p *Presenter) SetValue(value string) {
	if !p.state.Submitting && p.state.Mode == EnteringValue {
		p.state.Value = value
	}
}

func (p *Presenter) Submit() bool {
	if p.state.Submitting || p.state.Mode != EnteringValue {
		return false
	}
	control := targetControl(p.state.Operation)
	if control == "" {
		control = discoveryControl(p.state.Operation)
	}
	if !actionEnabled(p.state.Actions, control) {
		return false
	}
	value, err := p.parsedValue()
	if err != nil {
		p.fail(err)
		return false
	}
	switch p.state.Operation {
	case ShowExact, ShowKey:
		exact := p.state.Operation == ShowExact
		p.runQuery(func(ctx *middleware.Context) (any, error) { return p.app.Tags.Show(ctx, value, exact) })
		return true
	case Add, Remove:
		target, targetName, operation := p.state.Target, p.state.TargetName, p.state.Operation
		var mutation Result
		p.state.Submitting, p.state.Err = true, nil
		p.publish()
		return p.submit.Submit(func() error {
			var result tagging.Result
			var err error
			if operation == Add {
				result, err = p.app.Tags.Upsert(p.app.Context(), target, value)
			} else {
				result, err = p.app.Tags.Remove(p.app.Context(), target, value.Key)
			}
			if err == nil {
				mutation = Result{Target: target, TargetName: targetName, Tags: result.Tags.Sorted(), Changed: result.Changed}
			}
			return err
		}, func(err error) {
			p.state.Submitting = false
			if err != nil {
				p.state.Err = ui.PresentError(err)
				ui.ShowPresentation(p.dialogs, err)
				p.publish()
				return
			}
			p.state.Result = cloneResult(mutation)
			p.state.Mode, p.state.Err = Results, nil
			p.publish()
		})
	case Inspect, Summary:
		p.fail(errors.Invalidf("operation does not accept a value"))
		return false
	}
	return false
}

func (p *Presenter) Back() bool {
	if p.state.Submitting {
		return false
	}
	p.load.Invalidate()
	switch p.state.Mode {
	case Browsing:
		return false
	case PickingEntity:
		p.state.Mode, p.state.Err = PickingType, nil
	case EnteringValue:
		if !p.state.Target.IsZero() {
			p.state.Mode = PickingEntity
		} else {
			p.state.Mode = Browsing
		}
		p.state.Err = nil
	case Loading, Results, PickingType:
		if p.state.Mode == Results && p.state.Operation == ShowExact && p.state.Catalog != nil {
			p.state.Mode, p.state.Operation, p.state.Err = Results, Summary, nil
			p.state.Result = Result{Summaries: append([]tagging.Summary(nil), p.state.Catalog...)}
			p.applySummaryQuery()
		} else {
			p.state = State{Mode: Browsing, Actions: discoveryActions(p.state.Actions)}
		}
	}
	p.publish()
	return true
}

func (p *Presenter) runQuery(work func(*middleware.Context) (any, error)) {
	operation, target, targetName := p.state.Operation, p.state.Target, p.state.TargetName
	p.state.Mode, p.state.Err = Loading, nil
	p.publish()
	p.load.LoadContext(p.app.Context(), func(ctx context.Context) (any, error) { return work(p.app.ContextFrom(ctx)) }, func(r ui.LoadState[any]) {
		if r.Status == ui.Loading {
			return
		}
		if r.Status == ui.Failed {
			p.state.Mode, p.state.Err = Results, ui.PresentError(r.Err)
			ui.ShowPresentation(p.dialogs, r.Err)
			p.publish()
			return
		}
		result := Result{Target: target, TargetName: targetName}
		switch operation {
		case Inspect:
			result.Tags, _ = r.Value.(tag.Tags)
			result.Tags = result.Tags.Sorted()
			result.Inspected = true
		case ShowExact, ShowKey:
			result.References, _ = r.Value.([]tagging.Reference)
		case Summary:
			result.Summaries, _ = r.Value.([]tagging.Summary)
			p.state.Catalog = append([]tagging.Summary(nil), result.Summaries...)
			p.state.Query = ""
		case Add, Remove:
			p.state.Mode, p.state.Err = Results, errors.Internalf("unexpected tag query")
			p.publish()
			return
		}
		p.state.Result, p.state.Mode, p.state.Err = cloneResult(result), Results, nil
		if operation == Summary {
			p.applySummaryQuery()
		}
		p.publish()
	})
}

func (p *Presenter) parsedValue() (tag.Tag, error) {
	raw := strings.TrimSpace(p.state.Value)
	if p.state.Operation == Remove || p.state.Operation == ShowKey {
		return tag.New(raw, "")
	}
	return tag.Parse(raw)
}

func (p *Presenter) applyQuery() {
	query := strings.ToLower(strings.TrimSpace(p.state.Query))
	p.state.Visible = nil
	for _, option := range p.state.Entities {
		if query == "" || strings.Contains(strings.ToLower(option.Name+" "+option.Detail), query) {
			p.state.Visible = append(p.state.Visible, option)
		}
	}
}

func (p *Presenter) applySummaryQuery() {
	query := strings.ToLower(strings.TrimSpace(p.state.Query))
	p.state.VisibleSummaries = nil
	for _, summary := range p.state.Catalog {
		if query == "" || strings.Contains(strings.ToLower(summary.Tag), query) {
			p.state.VisibleSummaries = append(p.state.VisibleSummaries, summary)
		}
	}
	p.sortSummaries()
}

func (p *Presenter) sortSummaries() {
	column := p.state.SummarySort - 1
	if column < 0 {
		return
	}
	value := func(s tagging.Summary) any {
		switch column {
		case 0:
			return s.Tag
		case 1:
			return s.Total
		case 2:
			return s.Drinks
		case 3:
			return s.Ingredients
		case 4:
			return s.Inventory
		case 5:
			return s.Menus
		default:
			return s.Orders
		}
	}
	sort.SliceStable(p.state.VisibleSummaries, func(i, j int) bool {
		a, b := value(p.state.VisibleSummaries[i]), value(p.state.VisibleSummaries[j])
		less := false
		switch x := a.(type) {
		case string:
			less = x < b.(string)
		case int:
			less = x < b.(int)
		}
		if p.state.SummaryDescending {
			return !less && a != b
		}
		return less
	})
}

func (p *Presenter) loadEntities(ctx *middleware.Context, kind cedar.EntityType) ([]EntityOption, error) {
	switch kind {
	case entity.TypeDrink:
		items, err := paging.Collect(func(c paging.Cursor) (paging.Page[*drinksmodels.Drink], error) {
			return p.app.Drinks.List(ctx, drinks.ListRequest{Cursor: c})
		})
		if err != nil {
			return nil, err
		}
		out := make([]EntityOption, 0, len(items))
		for _, v := range items {
			out = append(out, EntityOption{UID: v.EntityUID(), Name: v.Name, Detail: fmt.Sprintf("%s • %s", v.Category, v.ID), Resource: v.CedarEntity()})
		}
		return out, nil
	case entity.TypeIngredient:
		items, err := paging.Collect(func(c paging.Cursor) (paging.Page[*ingredientsmodels.Ingredient], error) {
			return p.app.Ingredients.List(ctx, ingredients.ListRequest{Cursor: c})
		})
		if err != nil {
			return nil, err
		}
		out := make([]EntityOption, 0, len(items))
		for _, v := range items {
			out = append(out, EntityOption{UID: v.EntityUID(), Name: v.Name, Detail: fmt.Sprintf("%s • %s", v.Category, v.ID), Resource: v.CedarEntity()})
		}
		return out, nil
	case entity.TypeInventory:
		names, err := p.ingredientNames(ctx)
		if err != nil {
			return nil, err
		}
		items, err := paging.Collect(func(c paging.Cursor) (paging.Page[*inventorymodels.Inventory], error) {
			return p.app.Inventory.List(ctx, inventory.ListRequest{Cursor: c})
		})
		if err != nil {
			return nil, err
		}
		out := make([]EntityOption, 0, len(items))
		for _, v := range items {
			name := names[v.IngredientID]
			if name == "" {
				name = "Unknown ingredient"
			}
			out = append(out, EntityOption{UID: v.EntityUID(), Name: name, Detail: fmt.Sprintf("%s • %s", v.Amount, v.ID), Resource: v.CedarEntity()})
		}
		return out, nil
	case entity.TypeMenu:
		items, err := paging.Collect(func(c paging.Cursor) (paging.Page[*menusmodels.Menu], error) {
			return p.app.Menus.List(ctx, menus.ListRequest{Cursor: c})
		})
		if err != nil {
			return nil, err
		}
		out := make([]EntityOption, 0, len(items))
		for _, v := range items {
			out = append(out, EntityOption{UID: v.EntityUID(), Name: v.Name, Detail: fmt.Sprintf("%s • %s", v.Status, v.ID), Resource: v.CedarEntity()})
		}
		return out, nil
	case entity.TypeOrder:
		items, err := paging.Collect(func(c paging.Cursor) (paging.Page[*ordersmodels.Order], error) {
			return p.app.Orders.List(ctx, orders.ListRequest{Cursor: c})
		})
		if err != nil {
			return nil, err
		}
		out := make([]EntityOption, 0, len(items))
		for _, v := range items {
			out = append(out, EntityOption{UID: v.EntityUID(), Name: "Order " + v.ID.String(), Detail: fmt.Sprintf("%s • menu %s", v.Status, v.MenuID), Resource: v.CedarEntity()})
		}
		return out, nil
	default:
		return nil, errors.Invalidf("unsupported entity type")
	}
}

func (p *Presenter) ingredientNames(ctx *middleware.Context) (map[entity.IngredientID]string, error) {
	items, err := paging.Collect(func(c paging.Cursor) (paging.Page[*ingredientsmodels.Ingredient], error) {
		return p.app.Ingredients.List(ctx, ingredients.ListRequest{Cursor: c})
	})
	if err != nil {
		return nil, err
	}
	names := make(map[entity.IngredientID]string, len(items))
	for _, v := range items {
		names[v.ID] = v.Name
	}
	return names, nil
}
func supportedType(t cedar.EntityType) bool {
	return t == entity.TypeDrink || t == entity.TypeIngredient || t == entity.TypeInventory || t == entity.TypeMenu || t == entity.TypeOrder
}
func (p *Presenter) fail(err error) {
	p.state.Err = ui.PresentError(err)
	ui.ShowPresentation(p.dialogs, err)
	p.publish()
}
func (p *Presenter) publish() {
	if p.changed != nil {
		p.changed(cloneState(p.state))
	}
}
func cloneEntities(v []EntityOption) []EntityOption {
	result := append([]EntityOption(nil), v...)
	for i := range result {
		result[i].Actions = maps.Clone(result[i].Actions)
	}
	return result
}
func cloneResult(r Result) Result {
	r.Tags = append(tag.Tags(nil), r.Tags...)
	r.References = append([]tagging.Reference(nil), r.References...)
	r.Summaries = append([]tagging.Summary(nil), r.Summaries...)
	return r
}
func cloneState(s State) State {
	s.Entities = cloneEntities(s.Entities)
	s.Visible = cloneEntities(s.Visible)
	s.Result = cloneResult(s.Result)
	s.Catalog = append([]tagging.Summary(nil), s.Catalog...)
	s.VisibleSummaries = append([]tagging.Summary(nil), s.VisibleSummaries...)
	s.Actions = maps.Clone(s.Actions)
	return s
}

func actionMap(states []actions.State) map[actions.ID]actions.State {
	result := make(map[actions.ID]actions.State, len(states))
	for _, state := range states {
		result[state.ID] = state
	}
	return result
}
func actionEnabled(states map[actions.ID]actions.State, id actions.ID) bool {
	state, ok := states[id]
	return ok && state.Visible && state.Enabled
}
func discoveryControl(operation Operation) actions.ID {
	switch operation {
	case ShowExact, ShowKey:
		return tagging.ControlShow
	case Summary:
		return tagging.ControlSummary
	case Inspect, Add, Remove:
		return ""
	}
	return ""
}
func targetControl(operation Operation) actions.ID {
	switch operation {
	case Inspect:
		return tagging.ControlInspect
	case Add:
		return tagging.ControlTag
	case Remove:
		return tagging.ControlUntag
	case ShowExact, ShowKey, Summary:
		return ""
	}
	return ""
}
func discoveryActions(states map[actions.ID]actions.State) map[actions.ID]actions.State {
	return map[actions.ID]actions.State{tagging.ControlShow: states[tagging.ControlShow], tagging.ControlSummary: states[tagging.ControlSummary]}
}
