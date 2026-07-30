// Package fyne provides the bespoke retained-mode desktop presentation for tagging.
package fyne

import (
	"fmt"
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
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	ui "github.com/TheFellow/go-modular-monolith/pkg/fyne"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
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
	Mode       Mode
	Operation  Operation
	EntityType cedar.EntityType
	Entities   []EntityOption
	Visible    []EntityOption
	Query      string
	Target     cedar.EntityUID
	TargetName string
	Value      string
	Result     Result
	Submitting bool
	Err        error
}

type Dependencies struct {
	Executor   ui.Executor
	Dispatcher ui.Dispatcher
	Dialogs    ui.Dialogs
}

type Presenter struct {
	app     *app.Session
	dialogs ui.Dialogs
	load    *ui.LatestRequest[any]
	submit  *ui.Submission
	state   State
	changed func(State)
}

func NewPresenter(session *app.Session, deps Dependencies) *Presenter {
	return &Presenter{app: session, dialogs: deps.Dialogs, load: ui.NewLatestRequest[any](deps.Executor, deps.Dispatcher), submit: ui.NewSubmission(deps.Executor, deps.Dispatcher)}
}

func (p *Presenter) Observe(fn func(State)) { p.changed = fn; p.publish() }
func (p *Presenter) State() State           { return cloneState(p.state) }

func (p *Presenter) Start(operation Operation) {
	if p.state.Submitting {
		return
	}
	p.load.Invalidate()
	p.state = State{Operation: operation}
	switch operation {
	case Inspect, Add, Remove:
		p.state.Mode = PickingType
	case ShowExact, ShowKey:
		p.state.Mode = EnteringValue
	case Summary:
		p.runQuery(func() (any, error) { return p.app.Tags.Summary(p.app.Context()) })
	default:
		p.fail(apperrors.Invalidf("invalid tag operation"))
	}
	p.publish()
}

func (p *Presenter) SelectType(kind cedar.EntityType) {
	if p.state.Submitting || p.state.Mode != PickingType || !supportedType(kind) {
		return
	}
	p.state.EntityType, p.state.Mode, p.state.Err = kind, Loading, nil
	p.publish()
	p.load.Load(func() (any, error) { return p.loadEntities(kind) }, func(r ui.LoadState[any]) {
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
			err := apperrors.Internalf("unexpected entity catalog")
			p.state.Mode, p.state.Err = PickingType, ui.PresentError(err)
			ui.ShowPresentation(p.dialogs, err)
			p.publish()
			return
		}
		p.state.Entities, p.state.Query, p.state.Mode, p.state.Err = cloneEntities(options), "", PickingEntity, nil
		p.applyQuery()
		p.publish()
	})
}

func (p *Presenter) Search(query string) {
	if p.state.Submitting || p.state.Mode != PickingEntity {
		return
	}
	p.state.Query = query
	p.applyQuery()
	p.publish()
}

func (p *Presenter) SelectEntity(index int) {
	if p.state.Submitting || p.state.Mode != PickingEntity || index < 0 || index >= len(p.state.Visible) {
		return
	}
	selected := p.state.Visible[index]
	p.state.Target, p.state.TargetName, p.state.Value, p.state.Err = selected.UID, selected.Name, "", nil
	if p.state.Operation == Inspect {
		target := selected.UID
		p.runQuery(func() (any, error) { return p.app.Tags.List(p.app.Context(), target) })
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
	value, err := p.parsedValue()
	if err != nil {
		p.fail(err)
		return false
	}
	switch p.state.Operation {
	case ShowExact, ShowKey:
		exact := p.state.Operation == ShowExact
		p.runQuery(func() (any, error) { return p.app.Tags.Show(p.app.Context(), value, exact) })
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
		p.fail(apperrors.Invalidf("operation does not accept a value"))
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
		p.state = State{Mode: Browsing}
	}
	p.publish()
	return true
}

func (p *Presenter) runQuery(work func() (any, error)) {
	operation, target, targetName := p.state.Operation, p.state.Target, p.state.TargetName
	p.state.Mode, p.state.Err = Loading, nil
	p.publish()
	p.load.Load(work, func(r ui.LoadState[any]) {
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
		case Add, Remove:
			p.state.Mode, p.state.Err = Results, apperrors.Internalf("unexpected tag query")
			p.publish()
			return
		}
		p.state.Result, p.state.Mode, p.state.Err = cloneResult(result), Results, nil
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

func (p *Presenter) loadEntities(kind cedar.EntityType) ([]EntityOption, error) {
	ctx := p.app.Context()
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
			out = append(out, EntityOption{v.EntityUID(), v.Name, fmt.Sprintf("%s • %s", v.Category, v.ID)})
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
			out = append(out, EntityOption{v.EntityUID(), v.Name, fmt.Sprintf("%s • %s", v.Category, v.ID)})
		}
		return out, nil
	case entity.TypeInventory:
		names, err := p.ingredientNames()
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
			out = append(out, EntityOption{v.EntityUID(), name, fmt.Sprintf("%s • %s", v.Amount, v.ID)})
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
			out = append(out, EntityOption{v.EntityUID(), v.Name, fmt.Sprintf("%s • %s", v.Status, v.ID)})
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
			out = append(out, EntityOption{v.EntityUID(), "Order " + v.ID.String(), fmt.Sprintf("%s • menu %s", v.Status, v.MenuID)})
		}
		return out, nil
	default:
		return nil, apperrors.Invalidf("unsupported entity type")
	}
}

func (p *Presenter) ingredientNames() (map[entity.IngredientID]string, error) {
	items, err := paging.Collect(func(c paging.Cursor) (paging.Page[*ingredientsmodels.Ingredient], error) {
		return p.app.Ingredients.List(p.app.Context(), ingredients.ListRequest{Cursor: c})
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
func cloneEntities(v []EntityOption) []EntityOption { return append([]EntityOption(nil), v...) }
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
	return s
}
