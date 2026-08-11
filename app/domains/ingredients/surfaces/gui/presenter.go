// Package gui provides the Ingredients GUI presentation.
package gui

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	toolkit "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

type Mode uint8

const (
	Browse Mode = iota
	Viewing
	Create
	Edit
	Tags
)

type Form struct {
	Name, Description, Tags string
	Category                models.Category
	Unit                    measurement.Unit
	// ReplaceTags distinguishes an omitted complete set from an explicitly
	// empty set. Framework forms set it when the mutation includes tag input.
	ReplaceTags bool
}

type State struct {
	Status       toolkit.LoadStatus
	Items        []models.Ingredient
	Selected     *models.Ingredient
	Category     models.Category
	Expression   string
	Limit        int
	Cursor, Next paging.Cursor
	History      []paging.Cursor
	Mode         Mode
	Form         Form
	Err          error
	Submitting   bool
	Dirty        bool
	CanUpdate    bool
	CanDelete    bool
	CanTag       bool
	CanCreate    bool
	CanList      bool
	Actions      map[actions.ID]actions.State
	FormInstance uint64
}

type Presenter struct {
	app        *app.Session
	executor   toolkit.Executor
	dispatcher toolkit.Dispatcher
	dialogs    toolkit.Dialogs
	loads      *toolkit.LatestRequest[loadResult]
	mutation   *toolkit.Submission

	mu        sync.Mutex
	state     State
	changed   func(State)
	projector ingredients.ActionProjector
	sort      toolkit.TableSort
}
type loadResult struct {
	items []models.Ingredient
	next  paging.Cursor
}

func NewPresenter(session *app.Session, executor toolkit.Executor, dispatcher toolkit.Dispatcher, dialogs toolkit.Dialogs, projectors ...ingredients.ActionProjector) *Presenter {
	projector := ingredients.NewActionProjector()
	if len(projectors) > 0 {
		projector = projectors[0]
	}
	p := &Presenter{app: session, executor: executor, dispatcher: dispatcher, dialogs: dialogs, projector: projector, state: State{Limit: toolkit.PageLimit}}
	if err := p.permissionsForLocked(nil); err != nil {
		p.state.Err = toolkit.PresentError(err)
		toolkit.ShowPresentation(dialogs, err)
	}
	p.loads = toolkit.NewLatestRequest[loadResult](executor, dispatcher)
	p.mutation = toolkit.NewSubmission(executor, dispatcher)
	return p
}

func (p *Presenter) OnChange(changed func(State)) { p.changed = changed }

func (p *Presenter) Snapshot() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return cloneState(p.state)
}

func (p *Presenter) Load() {
	p.mu.Lock()
	p.state.Cursor, p.state.Next, p.state.History = "", "", nil
	p.mu.Unlock()
	p.loadPage(false)
}

func (p *Presenter) loadPage(appendPage bool) {
	p.mu.Lock()
	request := ingredients.ListRequest{Category: p.state.Category, Filter: p.state.Expression, Cursor: p.state.Cursor, Limit: p.state.Limit}
	p.mu.Unlock()
	p.loads.LoadContext(p.app.Context(), func(ctx context.Context) (loadResult, error) {
		page, err := p.app.Ingredients.List(p.app.ContextFrom(ctx), request)
		if err != nil {
			return loadResult{}, err
		}
		items := make([]models.Ingredient, 0, len(page.Items))
		for i, item := range page.Items {
			if item == nil {
				return loadResult{}, fmt.Errorf("ingredient %d missing", i)
			}
			items = append(items, *item)
		}
		return loadResult{items: items, next: page.Next}, nil
	}, func(result toolkit.LoadState[loadResult]) {
		p.mu.Lock()
		p.state.Status, p.state.Err = result.Status, toolkit.PresentError(result.Err)
		if result.Status == toolkit.Loaded {
			selected := selectedID(p.state.Selected)
			if appendPage {
				p.state.Items = append(p.state.Items, result.Value.items...)
			} else {
				p.state.Items = result.Value.items
			}
			p.sortItemsLocked()
			p.state.Next = result.Value.next
			p.state.Selected = findIngredient(p.state.Items, selected)
		}
		p.publishLocked()
		p.mu.Unlock()
		if result.Status == toolkit.Failed {
			toolkit.ShowPresentation(p.dialogs, result.Err)
		}
	})
}

func (p *Presenter) SortItems(column int, direction toolkit.SortDirection) {
	if column < 0 || column > 4 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sort = toolkit.TableSort{Column: column, Direction: direction}
	p.sortItemsLocked()
	p.publishLocked()
}

func (p *Presenter) sortItemsLocked() {
	toolkit.ApplyTableSort(p.state.Items, p.sort, func(column int, left, right models.Ingredient) int {
		switch column {
		case 0:
			return cmp.Compare(left.Name, right.Name)
		case 1:
			return cmp.Compare(left.Category, right.Category)
		case 2:
			return cmp.Compare(left.Unit, right.Unit)
		case 3:
			return cmp.Compare(left.Description, right.Description)
		case 4:
			return cmp.Compare(left.Tags.Canonical().String(), right.Tags.Canonical().String())
		}
		return 0
	})
}

func (p *Presenter) Filter(category models.Category, expression string, limits ...int) bool {
	limit := toolkit.PageLimit
	if len(limits) > 0 {
		limit = limits[0]
	}
	if limit <= 0 {
		p.mu.Lock()
		p.state.Err = toolkit.PresentError(fmt.Errorf("page size must be greater than zero"))
		p.publishLocked()
		p.mu.Unlock()
		return false
	}
	p.mu.Lock()
	p.state.Category, p.state.Expression, p.state.Limit = category, strings.TrimSpace(expression), limit
	p.state.Cursor, p.state.Next, p.state.History = "", "", nil
	p.mu.Unlock()
	p.loadPage(false)
	return true
}
func (p *Presenter) NextPage() {
	p.mu.Lock()
	if p.state.Next == "" || p.state.Status == toolkit.Loading {
		p.mu.Unlock()
		return
	}
	p.state.History = append(p.state.History, p.state.Cursor)
	p.state.Cursor = p.state.Next
	p.mu.Unlock()
	p.loadPage(true)
}
func (p *Presenter) PreviousPage() {
	p.mu.Lock()
	if len(p.state.History) == 0 || p.state.Status == toolkit.Loading {
		p.mu.Unlock()
		return
	}
	last := len(p.state.History) - 1
	p.state.Cursor = p.state.History[last]
	p.state.History = p.state.History[:last]
	p.mu.Unlock()
	p.loadPage(false)
}

func (p *Presenter) Select(id entity.IngredientID) {
	p.mu.Lock()
	p.state.Selected = findIngredient(p.state.Items, id)
	if p.state.Selected != nil {
		p.state.FormInstance++
		p.state.Form = formFromIngredient(p.state.Selected)
		if err := p.permissionsForLocked(p.state.Selected); err != nil {
			p.state.Err = toolkit.PresentError(err)
			toolkit.ShowPresentation(p.dialogs, err)
		}
		if p.state.CanUpdate {
			p.state.Mode = Edit
		} else {
			p.state.Mode = Viewing
		}
		p.state.Dirty = false
	}
	p.publishLocked()
	p.mu.Unlock()
}

// Back returns to the exact catalog state from which detail was opened.
func (p *Presenter) Back() { p.leaveDetail(false) }

// ResetList returns to an unfiltered first page, for breadcrumbs and main navigation.
func (p *Presenter) ResetList() { p.leaveDetail(true) }

func (p *Presenter) leaveDetail(reset bool) {
	p.mu.Lock()
	if p.state.Submitting {
		p.mu.Unlock()
		return
	}
	proceed := func() {
		p.mu.Lock()
		if reset {
			p.state.Category, p.state.Expression, p.state.Limit = "", "", toolkit.PageLimit
			p.state.Cursor, p.state.Next, p.state.History = "", "", nil
		}
		p.state.Mode, p.state.Dirty, p.state.Err = Browse, false, nil
		p.publishLocked()
		p.mu.Unlock()
		if reset {
			p.Load()
		}
	}
	dirty := p.state.Dirty
	p.mu.Unlock()
	if dirty {
		if p.dialogs == nil {
			return
		}
		p.dialogs.Confirm("Discard changes?", "Discard unsaved ingredient changes?", func(ok bool) {
			if ok {
				proceed()
			}
		})
		return
	}
	proceed()
}

func (p *Presenter) StartCreate() {
	p.mu.Lock()
	if !p.actionEnabledLocked(ingredients.ControlCreate) {
		p.mu.Unlock()
		return
	}
	p.state.Mode, p.state.Form, p.state.Err = Create, Form{}, nil
	p.publishLocked()
	p.mu.Unlock()
}

func (p *Presenter) StartEdit() {
	p.mu.Lock()
	if selected := p.state.Selected; selected != nil {
		if err := p.permissionsForLocked(selected); err != nil {
			p.state.Err = toolkit.PresentError(err)
			toolkit.ShowPresentation(p.dialogs, err)
		}
		if p.actionEnabledLocked(ingredients.ControlEdit) {
			p.state.Mode = Edit
		} else {
			p.state.Mode = Viewing
		}
		p.state.Form = formFromIngredient(selected)
		p.state.Err = nil
	}
	p.publishLocked()
	p.mu.Unlock()
}

func (p *Presenter) StartTags() {
	p.mu.Lock()
	if selected := p.state.Selected; selected != nil && p.actionEnabledLocked(ingredients.ControlTags) {
		p.state.Mode = Tags
		p.state.Form = Form{Tags: selected.Tags.Canonical().String()}
		p.state.Err = nil
	}
	p.publishLocked()
	p.mu.Unlock()
}

func (p *Presenter) Cancel() {
	p.mu.Lock()
	if !p.state.Submitting && p.state.Mode == Edit && p.state.Selected != nil {
		p.state.Form = formFromIngredient(p.state.Selected)
		p.state.FormInstance++
		p.state.Dirty, p.state.Err = false, nil
	} else if !p.state.Submitting {
		p.state.Mode, p.state.Err, p.state.Dirty = Browse, nil, false
	}
	p.publishLocked()
	p.mu.Unlock()
}

func (p *Presenter) SetForm(form Form) {
	p.mu.Lock()
	p.state.Form = form
	p.state.Dirty = p.state.Selected != nil && !reflect.DeepEqual(form, formFromIngredient(p.state.Selected))
	p.publishLocked()
	p.mu.Unlock()
}

func (p *Presenter) Submit(form Form) bool {
	p.mu.Lock()
	mode, selected := p.state.Mode, p.state.Selected
	p.state.Form = form
	if mode == Viewing {
		p.mu.Unlock()
		return false
	}
	if mode == Edit && selected != nil {
		p.state.Dirty = !reflect.DeepEqual(form, formFromIngredient(selected))
	}
	if mode == Create && !p.actionEnabledLocked(ingredients.ControlCreate) ||
		mode == Edit && (!p.actionEnabledLocked(ingredients.ControlEdit) || !p.state.Dirty) ||
		mode == Tags && !p.actionEnabledLocked(ingredients.ControlTags) {
		p.mu.Unlock()
		return false
	}
	err := validateForm(mode, form)
	if err != nil {
		p.state.Err = toolkit.PresentError(err)
		p.publishLocked()
		p.mu.Unlock()
		return false
	}
	p.state.Err = nil
	p.state.Submitting = true
	p.publishLocked()
	p.mu.Unlock()

	accepted := p.mutation.Submit(func() error {
		category := models.Category(strings.TrimSpace(string(form.Category)))
		unit := measurement.Unit(strings.TrimSpace(string(form.Unit)))
		var desired *tag.Tags
		if form.ReplaceTags {
			values, parseErr := tag.ParseCollection(form.Tags)
			if parseErr != nil {
				return parseErr
			}
			desired = &values
		}
		switch mode {
		case Create:
			_, err = app.RunTaggedMutation(p.app.App, p.app.Context(), desired, func(ctx *middleware.Context) (*models.Ingredient, error) {
				return p.app.Ingredients.Create(ctx, &models.Ingredient{Name: strings.TrimSpace(form.Name), Category: category, Unit: unit, Description: strings.TrimSpace(form.Description)})
			})
		case Edit:
			if selected == nil {
				return errors.Invalidf("ingredient is required")
			}
			_, err = app.RunTaggedMutation(p.app.App, p.app.Context(), desired, func(ctx *middleware.Context) (*models.Ingredient, error) {
				return p.app.Ingredients.Update(ctx, &models.Ingredient{ID: selected.ID, Name: strings.TrimSpace(form.Name), Category: category, Unit: unit, Description: strings.TrimSpace(form.Description)})
			})
		case Tags:
			if selected == nil {
				return errors.Invalidf("ingredient is required")
			}
			var desired tag.Tags
			desired, err = tag.ParseCollection(form.Tags)
			if err == nil {
				_, err = p.app.Tags.Replace(p.app.Context(), selected.EntityUID(), desired)
			}
		case Browse, Viewing:
			err = errors.FailedPreconditionf("ingredient form is not active")
		}
		return err
	}, func(err error) {
		p.mu.Lock()
		p.state.Submitting = false
		p.state.Err = toolkit.PresentError(err)
		stayDetail := err == nil && mode == Edit && selected != nil
		p.publishLocked()
		p.mu.Unlock()
		toolkit.ShowPresentation(p.dialogs, err)
		if stayDetail {
			updated, getErr := p.app.Ingredients.Get(p.app.Context(), selected.ID)
			p.mu.Lock()
			if getErr != nil {
				p.state.Err = toolkit.PresentError(getErr)
			} else {
				p.state.Selected = updated
				p.state.Form = formFromIngredient(updated)
				p.state.Dirty = false
				for i := range p.state.Items {
					if p.state.Items[i].ID == updated.ID {
						p.state.Items[i] = *updated
					}
				}
			}
			p.publishLocked()
			p.mu.Unlock()
		} else if err == nil {
			p.mu.Lock()
			p.state.Mode, p.state.Dirty = Browse, false
			p.publishLocked()
			p.mu.Unlock()
			p.Load()
		}
	})
	if !accepted {
		p.mu.Lock()
		p.state.Submitting = p.mutation.Active()
		p.publishLocked()
		p.mu.Unlock()
	}
	return accepted
}

func (p *Presenter) RequestDelete() {
	p.RequestRetire("", "")
}

func (p *Presenter) RequestRetire(replacementID, replacementRatio string) {
	p.mu.Lock()
	target := p.state.Selected
	allowed := p.actionEnabledLocked(ingredients.ControlDelete)
	p.mu.Unlock()
	if target == nil || !allowed {
		return
	}
	retirement := models.Retirement{}
	replacementID = strings.TrimSpace(replacementID)
	replacementRatio = strings.TrimSpace(replacementRatio)
	if replacementID != "" {
		parsed, err := entity.ParseIngredientID(replacementID)
		if err != nil {
			p.presentRetirementError(err)
			return
		}
		retirement.ReplacementID = parsed
		if replacementRatio != "" {
			ratio, err := strconv.ParseFloat(replacementRatio, 64)
			if err != nil {
				p.presentRetirementError(errors.Invalidf("invalid replacement ratio %q", replacementRatio))
				return
			}
			retirement.Ratio = ratio
		}
	}
	p.executor.Execute(func() {
		count, err := p.countDrinksUsing(target.ID)
		p.dispatcher.Dispatch(func() {
			if err != nil {
				p.mu.Lock()
				p.state.Err = toolkit.PresentError(err)
				p.publishLocked()
				p.mu.Unlock()
				toolkit.ShowPresentation(p.dialogs, err)
				return
			}
			message := fmt.Sprintf("Retire %q?", target.Name)
			if count > 0 {
				message = fmt.Sprintf("Retire %q?\n\nThis will mark %d dependent drink(s) for review and make their menu items unavailable.", target.Name, count)
			}
			p.dialogs.Confirm("Retire Ingredient", message, func(confirmed bool) {
				if confirmed {
					p.retire(target.ID, retirement)
				}
			})
		})
	})
}

func (p *Presenter) presentRetirementError(err error) {
	p.mu.Lock()
	p.state.Err = toolkit.PresentError(err)
	p.publishLocked()
	p.mu.Unlock()
	toolkit.ShowPresentation(p.dialogs, err)
}

func (p *Presenter) retire(id entity.IngredientID, retirement models.Retirement) bool {
	p.mu.Lock()
	p.state.Submitting = true
	p.publishLocked()
	p.mu.Unlock()
	accepted := p.mutation.Submit(func() error {
		_, err := p.app.Ingredients.Retire(p.app.Context(), id, retirement)
		return err
	}, func(err error) {
		p.mu.Lock()
		p.state.Submitting, p.state.Err = false, toolkit.PresentError(err)
		if err == nil {
			p.state.Mode, p.state.Dirty, p.state.Selected = Browse, false, nil
		}
		p.publishLocked()
		p.mu.Unlock()
		toolkit.ShowPresentation(p.dialogs, err)
		if err == nil {
			p.Load()
		}
	})
	if !accepted {
		p.mu.Lock()
		p.state.Submitting = p.mutation.Active()
		p.publishLocked()
		p.mu.Unlock()
	}
	return accepted
}

func (p *Presenter) countDrinksUsing(ingredientID entity.IngredientID) (int, error) {
	count := 0
	request := drinks.ListRequest{Limit: paging.DefaultLimit}
	for {
		page, err := p.app.Drinks.List(p.app.Context(), request)
		if err != nil {
			return 0, err
		}
		count += countDrinksUsing(page.Items, ingredientID)
		if page.Next == "" {
			return count, nil
		}
		request.Cursor = page.Next
	}
}

func validateForm(mode Mode, form Form) error {
	if mode == Tags {
		_, err := tag.ParseCollection(form.Tags)
		return err
	}
	name := strings.TrimSpace(form.Name)
	if name == "" {
		return errors.Invalidf("name is required")
	}
	if utf8.RuneCountInString(name) > 100 {
		return errors.Invalidf("name must be at most 100 characters")
	}
	if err := form.Category.Validate(); err != nil {
		return err
	}
	if err := form.Unit.Validate(); err != nil {
		return err
	}
	if utf8.RuneCountInString(strings.TrimSpace(form.Description)) > 500 {
		return errors.Invalidf("description must be at most 500 characters")
	}
	return nil
}

func (p *Presenter) publishLocked() {
	if p.changed != nil {
		p.changed(cloneState(p.state))
	}
}

func (p *Presenter) permissionsForLocked(ingredient *models.Ingredient) error {
	p.state.Actions = nil
	p.state.CanList, p.state.CanCreate, p.state.CanUpdate, p.state.CanDelete, p.state.CanTag = false, false, false, false, false
	states, err := p.projector.Project(p.app.Context(), p.app.Context().Principal(), ingredient)
	if err != nil {
		return err
	}
	p.state.Actions = make(map[actions.ID]actions.State, len(states))
	for _, state := range states {
		p.state.Actions[state.ID] = state
	}
	p.state.CanList = p.state.Actions[ingredients.ControlList].Visible
	p.state.CanCreate = p.state.Actions[ingredients.ControlCreate].Visible
	p.state.CanUpdate = p.state.Actions[ingredients.ControlEdit].Visible
	p.state.CanDelete = p.state.Actions[ingredients.ControlDelete].Visible
	p.state.CanTag = p.state.Actions[ingredients.ControlTags].Visible
	return nil
}

func (p *Presenter) actionEnabledLocked(id actions.ID) bool {
	state, ok := p.state.Actions[id]
	return ok && state.Visible && state.Enabled
}

func cloneState(state State) State {
	state.Items = append([]models.Ingredient(nil), state.Items...)
	state.History = append([]paging.Cursor(nil), state.History...)
	actionsCopy := make(map[actions.ID]actions.State, len(state.Actions))
	maps.Copy(actionsCopy, state.Actions)
	state.Actions = actionsCopy
	if state.Selected != nil {
		selected := *state.Selected
		state.Selected = &selected
	}
	return state
}

func formFromIngredient(value *models.Ingredient) Form {
	if value == nil {
		return Form{}
	}
	return Form{Name: value.Name, Category: value.Category, Unit: value.Unit, Description: value.Description, Tags: value.Tags.Canonical().String(), ReplaceTags: true}
}

func selectedID(value *models.Ingredient) entity.IngredientID {
	if value == nil {
		return entity.IngredientID{}
	}
	return value.ID
}

func findIngredient(items []models.Ingredient, id entity.IngredientID) *models.Ingredient {
	for i := range items {
		if items[i].ID == id {
			value := items[i]
			return &value
		}
	}
	return nil
}

func countDrinksUsing(values []*drinksmodels.Drink, ingredientID entity.IngredientID) int {
	count := 0
	for _, drink := range values {
		if drink == nil {
			continue
		}
		for _, recipeIngredient := range drink.Recipe.Ingredients {
			if recipeIngredient.IngredientID == ingredientID || slices.Contains(recipeIngredient.Substitutes, ingredientID) {
				count++
				break
			}
		}
	}
	return count
}
