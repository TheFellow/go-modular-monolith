// Package fyne provides the Fyne-native Ingredients presentation.
package fyne

import (
	"fmt"
	"slices"
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
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	fyneui "github.com/TheFellow/go-modular-monolith/pkg/fyne"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
)

type Mode uint8

const (
	Browse Mode = iota
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
	Status       fyneui.LoadStatus
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
}

type Presenter struct {
	app        *app.Session
	executor   fyneui.Executor
	dispatcher fyneui.Dispatcher
	dialogs    fyneui.Dialogs
	loads      *fyneui.LatestRequest[loadResult]
	mutation   *fyneui.Submission

	mu      sync.Mutex
	state   State
	changed func(State)
}
type loadResult struct {
	items []models.Ingredient
	next  paging.Cursor
}

func NewPresenter(session *app.Session, executor fyneui.Executor, dispatcher fyneui.Dispatcher, dialogs fyneui.Dialogs) *Presenter {
	p := &Presenter{app: session, executor: executor, dispatcher: dispatcher, dialogs: dialogs, state: State{Limit: paging.DefaultLimit}}
	p.loads = fyneui.NewLatestRequest[loadResult](executor, dispatcher)
	p.mutation = fyneui.NewSubmission(executor, dispatcher)
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
	request := ingredients.ListRequest{Category: p.state.Category, Filter: p.state.Expression, Cursor: p.state.Cursor, Limit: p.state.Limit}
	p.mu.Unlock()
	p.loads.Load(func() (loadResult, error) {
		page, err := p.app.Ingredients.List(p.app.Context(), request)
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
	}, func(result fyneui.LoadState[loadResult]) {
		p.mu.Lock()
		p.state.Status, p.state.Err = result.Status, fyneui.PresentError(result.Err)
		if result.Status == fyneui.Loaded {
			selected := selectedID(p.state.Selected)
			p.state.Items, p.state.Next = result.Value.items, result.Value.next
			p.state.Selected = findIngredient(p.state.Items, selected)
			if p.state.Selected == nil && len(p.state.Items) > 0 {
				value := p.state.Items[0]
				p.state.Selected = &value
			}
		}
		p.publishLocked()
		p.mu.Unlock()
		if result.Status == fyneui.Failed {
			fyneui.ShowPresentation(p.dialogs, result.Err)
		}
	})
}

func (p *Presenter) Filter(category models.Category, expression string, limits ...int) bool {
	limit := paging.DefaultLimit
	if len(limits) > 0 {
		limit = limits[0]
	}
	if limit <= 0 {
		p.mu.Lock()
		p.state.Err = fyneui.PresentError(fmt.Errorf("page size must be greater than zero"))
		p.publishLocked()
		p.mu.Unlock()
		return false
	}
	p.mu.Lock()
	p.state.Category, p.state.Expression, p.state.Limit = category, strings.TrimSpace(expression), limit
	p.state.Cursor, p.state.Next, p.state.History = "", "", nil
	p.mu.Unlock()
	p.Load()
	return true
}
func (p *Presenter) NextPage() {
	p.mu.Lock()
	if p.state.Next == "" || p.state.Status == fyneui.Loading {
		p.mu.Unlock()
		return
	}
	p.state.History = append(p.state.History, p.state.Cursor)
	p.state.Cursor = p.state.Next
	p.mu.Unlock()
	p.Load()
}
func (p *Presenter) PreviousPage() {
	p.mu.Lock()
	if len(p.state.History) == 0 || p.state.Status == fyneui.Loading {
		p.mu.Unlock()
		return
	}
	last := len(p.state.History) - 1
	p.state.Cursor = p.state.History[last]
	p.state.History = p.state.History[:last]
	p.mu.Unlock()
	p.Load()
}

func (p *Presenter) Select(id entity.IngredientID) {
	p.mu.Lock()
	p.state.Selected = findIngredient(p.state.Items, id)
	p.publishLocked()
	p.mu.Unlock()
}

func (p *Presenter) StartCreate() {
	p.mu.Lock()
	p.state.Mode, p.state.Form, p.state.Err = Create, Form{}, nil
	p.publishLocked()
	p.mu.Unlock()
}

func (p *Presenter) StartEdit() {
	p.mu.Lock()
	if selected := p.state.Selected; selected != nil {
		p.state.Mode = Edit
		p.state.Form = Form{Name: selected.Name, Category: selected.Category, Unit: selected.Unit, Description: selected.Description, Tags: selected.Tags.Canonical().String(), ReplaceTags: true}
		p.state.Err = nil
	}
	p.publishLocked()
	p.mu.Unlock()
}

func (p *Presenter) StartTags() {
	p.mu.Lock()
	if selected := p.state.Selected; selected != nil {
		p.state.Mode = Tags
		p.state.Form = Form{Tags: selected.Tags.Canonical().String()}
		p.state.Err = nil
	}
	p.publishLocked()
	p.mu.Unlock()
}

func (p *Presenter) Cancel() {
	p.mu.Lock()
	if !p.state.Submitting {
		p.state.Mode, p.state.Err = Browse, nil
	}
	p.publishLocked()
	p.mu.Unlock()
}

func (p *Presenter) Submit(form Form) bool {
	p.mu.Lock()
	mode, selected := p.state.Mode, p.state.Selected
	p.state.Form = form
	err := validateForm(mode, form)
	if err != nil {
		p.state.Err = fyneui.PresentError(err)
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
		switch mode { //nolint:exhaustive // browse mode does not submit a mutation.
		case Create:
			_, err = app.RunTaggedMutation(p.app.App, p.app.Context(), desired, func(ctx *middleware.Context) (*models.Ingredient, error) {
				return p.app.Ingredients.Create(ctx, &models.Ingredient{Name: strings.TrimSpace(form.Name), Category: category, Unit: unit, Description: strings.TrimSpace(form.Description)})
			})
		case Edit:
			if selected == nil {
				return fmt.Errorf("ingredient is required")
			}
			_, err = app.RunTaggedMutation(p.app.App, p.app.Context(), desired, func(ctx *middleware.Context) (*models.Ingredient, error) {
				return p.app.Ingredients.Update(ctx, &models.Ingredient{ID: selected.ID, Name: strings.TrimSpace(form.Name), Category: category, Unit: unit, Description: strings.TrimSpace(form.Description)})
			})
		case Tags:
			if selected == nil {
				return fmt.Errorf("ingredient is required")
			}
			var desired tag.Tags
			desired, err = tag.ParseCollection(form.Tags)
			if err == nil {
				_, err = p.app.Tags.Replace(p.app.Context(), selected.EntityUID(), desired)
			}
		default:
			err = fmt.Errorf("ingredient form is not active")
		}
		return err
	}, func(err error) {
		p.mu.Lock()
		p.state.Submitting = false
		p.state.Err = fyneui.PresentError(err)
		if err == nil {
			p.state.Mode = Browse
		}
		p.publishLocked()
		p.mu.Unlock()
		fyneui.ShowPresentation(p.dialogs, err)
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

func (p *Presenter) RequestDelete() {
	p.mu.Lock()
	target := p.state.Selected
	p.mu.Unlock()
	if target == nil {
		return
	}
	p.executor.Execute(func() {
		count, err := p.countDrinksUsing(target.ID)
		p.dispatcher.Dispatch(func() {
			if err != nil {
				p.mu.Lock()
				p.state.Err = fyneui.PresentError(err)
				p.publishLocked()
				p.mu.Unlock()
				fyneui.ShowPresentation(p.dialogs, err)
				return
			}
			message := fmt.Sprintf("Delete %q?", target.Name)
			if count > 0 {
				message = fmt.Sprintf("Delete %q?\n\nThis will also delete %d drink(s) that use this ingredient.", target.Name, count)
			}
			p.dialogs.Confirm("Delete Ingredient", message, func(confirmed bool) {
				if confirmed {
					p.delete(target.ID)
				}
			})
		})
	})
}

func (p *Presenter) delete(id entity.IngredientID) bool {
	p.mu.Lock()
	p.state.Submitting = true
	p.publishLocked()
	p.mu.Unlock()
	accepted := p.mutation.Submit(func() error {
		_, err := p.app.Ingredients.Delete(p.app.Context(), id)
		return err
	}, func(err error) {
		p.mu.Lock()
		p.state.Submitting, p.state.Err = false, fyneui.PresentError(err)
		p.publishLocked()
		p.mu.Unlock()
		fyneui.ShowPresentation(p.dialogs, err)
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
		return apperrors.Invalidf("name is required")
	}
	if utf8.RuneCountInString(name) > 100 {
		return apperrors.Invalidf("name must be at most 100 characters")
	}
	if err := form.Category.Validate(); err != nil {
		return err
	}
	if err := form.Unit.Validate(); err != nil {
		return err
	}
	if utf8.RuneCountInString(strings.TrimSpace(form.Description)) > 500 {
		return apperrors.Invalidf("description must be at most 500 characters")
	}
	return nil
}

func (p *Presenter) publishLocked() {
	if p.changed != nil {
		p.changed(cloneState(p.state))
	}
}

func cloneState(state State) State {
	state.Items = append([]models.Ingredient(nil), state.Items...)
	state.History = append([]paging.Cursor(nil), state.History...)
	if state.Selected != nil {
		selected := *state.Selected
		state.Selected = &selected
	}
	return state
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
