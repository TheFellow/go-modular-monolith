// Package fyne implements the retained-mode desktop surface for drinks.
package fyne

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	domain "github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsdomain "github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	menusdomain "github.com/TheFellow/go-modular-monolith/app/domains/menus"
	menusmodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	ui "github.com/TheFellow/go-modular-monolith/pkg/fyne"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
)

type Mode uint8

const (
	Browsing Mode = iota
	Creating
	Editing
	Tagging
)

type Filter struct {
	Name, Category, Glass, Expression string
	Limit                             int
}
type RecipeRow struct {
	Ingredient  entity.IngredientID
	Amount      string
	Unit        measurement.Unit
	Optional    bool
	Substitutes []entity.IngredientID
}
type IngredientOption struct {
	ID   entity.IngredientID
	Name string
}
type Form struct {
	Name, Category, Glass, Description string
	Recipe                             []RecipeRow
	Steps, Garnish, Tags               string
	ReplaceTags                        bool
}
type State struct {
	Mode                Mode
	FormInstance        uint64
	Loading, Submitting bool
	Items               []*models.Drink
	Selected            *models.Drink
	Filter              Filter
	Cursor, Next        paging.Cursor
	History             []paging.Cursor
	Form                Form
	Ingredients         []IngredientOption
	Err                 error
}
type Dependencies struct {
	Executor   ui.Executor
	Dispatcher ui.Dispatcher
	Dialogs    ui.Dialogs
}
type drinkCatalog struct {
	drinks      []*models.Drink
	ingredients []IngredientOption
	next        paging.Cursor
}

type Presenter struct {
	app              *app.Session
	dialogs          ui.Dialogs
	load             *ui.LatestRequest[drinkCatalog]
	ingredients      *ui.LatestRequest[[]IngredientOption]
	deleteCheck      *ui.LatestRequest[int]
	submit           *ui.Submission
	state            State
	changed          func(State)
	confirmingDelete bool
}

func NewPresenter(session *app.Session, d Dependencies) *Presenter {
	p := &Presenter{app: session, dialogs: d.Dialogs, state: State{Filter: Filter{Limit: paging.DefaultLimit}}}
	p.load = ui.NewLatestRequest[drinkCatalog](d.Executor, d.Dispatcher)
	p.ingredients = ui.NewLatestRequest[[]IngredientOption](d.Executor, d.Dispatcher)
	p.deleteCheck = ui.NewLatestRequest[int](d.Executor, d.Dispatcher)
	p.submit = ui.NewSubmission(d.Executor, d.Dispatcher)
	return p
}
func (p *Presenter) Observe(fn func(State)) { p.changed = fn; p.publish() }
func (p *Presenter) State() State           { return cloneState(p.state) }
func (p *Presenter) SetFilter(f Filter) bool {
	if f.Limit < 0 {
		p.fail(apperrors.Invalidf("page size must be greater than zero"))
		return false
	}
	if f.Limit == 0 {
		f.Limit = paging.DefaultLimit
	}
	f.Name, f.Expression = strings.TrimSpace(f.Name), strings.TrimSpace(f.Expression)
	p.state.Filter, p.state.Cursor, p.state.Next, p.state.History, p.state.Err = f, "", "", nil, nil
	p.publish()
	return true
}
func (p *Presenter) Refresh() {
	f := p.state.Filter
	cursor := p.state.Cursor
	p.load.Load(func() (drinkCatalog, error) {
		page, err := p.app.Drinks.List(p.app.Context(), domain.ListRequest{Name: f.Name, Category: models.DrinkCategory(strings.TrimSpace(f.Category)), Glass: models.GlassType(strings.TrimSpace(f.Glass)), Filter: f.Expression, Cursor: cursor, Limit: f.Limit})
		if err != nil {
			return drinkCatalog{}, err
		}
		ingredients, err := p.listIngredients()
		return drinkCatalog{drinks: page.Items, ingredients: ingredients, next: page.Next}, err
	}, func(r ui.LoadState[drinkCatalog]) {
		p.state.Loading = r.Status == ui.Loading
		if r.Status == ui.Failed {
			p.state.Err = ui.PresentError(r.Err)
			ui.ShowPresentation(p.dialogs, r.Err)
		}
		if r.Status == ui.Loaded {
			p.state.Err = nil
			p.state.Items = cloneDrinks(r.Value.drinks)
			p.state.Next = r.Value.next
			p.state.Ingredients = append([]IngredientOption(nil), r.Value.ingredients...)
			p.reselect()
		}
		p.publish()
	})
}
func (p *Presenter) NextPage() {
	if p.state.Next == "" || p.state.Loading {
		return
	}
	p.state.History = append(p.state.History, p.state.Cursor)
	p.state.Cursor = p.state.Next
	p.Refresh()
}
func (p *Presenter) PreviousPage() {
	if len(p.state.History) == 0 || p.state.Loading {
		return
	}
	last := len(p.state.History) - 1
	p.state.Cursor = p.state.History[last]
	p.state.History = p.state.History[:last]
	p.Refresh()
}
func (p *Presenter) Select(i int) {
	if i < 0 || i >= len(p.state.Items) {
		p.state.Selected = nil
	} else {
		p.state.Selected = cloneDrink(p.state.Items[i])
	}
	p.publish()
}
func (p *Presenter) StartCreate() {
	p.state.FormInstance++
	p.state.Mode, p.state.Form, p.state.Err = Creating, Form{Recipe: []RecipeRow{{Unit: measurement.UnitOz}}, ReplaceTags: true}, nil
	p.publish()
	p.loadIngredients()
}
func (p *Presenter) StartEdit() {
	if p.state.Selected == nil {
		return
	}
	p.state.FormInstance++
	p.state.Mode, p.state.Form, p.state.Err = Editing, formFromDrink(p.state.Selected), nil
	p.publish()
	p.loadIngredients()
}
func (p *Presenter) StartTags() {
	if p.state.Selected == nil {
		return
	}
	p.state.FormInstance++
	values, _ := tag.FormatCollection(p.state.Selected.Tags)
	p.state.Mode, p.state.Form, p.state.Err = Tagging, Form{Tags: values}, nil
	p.publish()
}
func (p *Presenter) Cancel() {
	if p.submit.Active() {
		return
	}
	p.state.Mode, p.state.Err = Browsing, nil
	p.publish()
}
func (p *Presenter) SetForm(f Form) { p.state.Form = cloneForm(f); p.publish() }
func (p *Presenter) loadIngredients() {
	p.ingredients.Load(func() ([]IngredientOption, error) {
		return p.listIngredients()
	}, func(r ui.LoadState[[]IngredientOption]) {
		p.state.Loading = r.Status == ui.Loading
		if r.Status == ui.Failed {
			p.state.Err = ui.PresentError(r.Err)
			ui.ShowPresentation(p.dialogs, r.Err)
		}
		if r.Status == ui.Loaded {
			p.state.Ingredients = append([]IngredientOption(nil), r.Value...)
			p.state.Err = nil
		}
		p.publish()
	})
}
func (p *Presenter) listIngredients() ([]IngredientOption, error) {
	items, err := paging.Collect(func(cursor paging.Cursor) (paging.Page[*ingredientsmodels.Ingredient], error) {
		return p.app.Ingredients.List(p.app.Context(), ingredientsdomain.ListRequest{Cursor: cursor})
	})
	if err != nil {
		return nil, err
	}
	out := make([]IngredientOption, 0, len(items))
	for _, item := range items {
		out = append(out, IngredientOption{ID: item.ID, Name: item.Name})
	}
	return out, nil
}

func (p *Presenter) Save() bool {
	switch p.state.Mode {
	case Tagging:
		return p.saveTags()
	case Creating, Editing:
	default:
		p.fail(apperrors.Invalidf("no drink form is active"))
		return false
	}
	drink, err := p.formDrink()
	if err != nil {
		p.fail(err)
		return false
	}
	mode := p.state.Mode
	var desired *tag.Tags
	if p.state.Form.ReplaceTags {
		values, parseErr := tag.ParseCollection(p.state.Form.Tags)
		if parseErr != nil {
			p.fail(parseErr)
			return false
		}
		desired = &values
	}
	return p.mutate(func() error {
		if mode == Creating {
			_, err = app.RunTaggedMutation(p.app.App, p.app.Context(), desired, func(ctx *middleware.Context) (*models.Drink, error) { return p.app.Drinks.Create(ctx, drink) })
		} else {
			_, err = app.RunTaggedMutation(p.app.App, p.app.Context(), desired, func(ctx *middleware.Context) (*models.Drink, error) { return p.app.Drinks.Update(ctx, drink) })
		}
		return err
	})
}
func (p *Presenter) Delete() {
	target := cloneDrink(p.state.Selected)
	if target == nil || p.dialogs == nil || p.confirmingDelete || p.submit.Active() {
		return
	}
	p.confirmingDelete = true
	p.deleteCheck.Load(func() (int, error) {
		items, err := paging.Collect(func(cursor paging.Cursor) (paging.Page[*menusmodels.Menu], error) {
			return p.app.Menus.List(p.app.Context(), menusdomain.ListRequest{Cursor: cursor})
		})
		return countMenusWithDrink(items, target.ID), err
	}, func(r ui.LoadState[int]) {
		if r.Status == ui.Failed {
			p.confirmingDelete = false
			p.fail(r.Err)
		}
		if r.Status == ui.Loaded {
			message := fmt.Sprintf("Delete %q?", target.Name)
			if r.Value > 0 {
				message = fmt.Sprintf("Delete %q?\n\nThis drink appears on %d menu(s) and will be removed from them.", target.Name, r.Value)
			}
			p.dialogs.Confirm("Delete drink", message, func(ok bool) {
				p.confirmingDelete = false
				if ok {
					p.mutate(func() error { _, err := p.app.Drinks.Delete(p.app.Context(), target.ID); return err })
				}
			})
		}
	})
}
func countMenusWithDrink(items []*menusmodels.Menu, id entity.DrinkID) int {
	n := 0
	for _, menu := range items {
		if menu == nil {
			continue
		}
		for _, item := range menu.Items {
			if item.DrinkID == id {
				n++
				break
			}
		}
	}
	return n
}
func (p *Presenter) saveTags() bool {
	target := cloneDrink(p.state.Selected)
	if target == nil {
		p.fail(apperrors.Invalidf("drink not selected"))
		return false
	}
	tags, err := tag.ParseCollection(p.state.Form.Tags)
	if err != nil {
		p.fail(err)
		return false
	}
	return p.mutate(func() error { _, err := p.app.Tags.Replace(p.app.Context(), target.EntityUID(), tags); return err })
}
func (p *Presenter) mutate(work func() error) bool {
	if p.submit.Active() {
		return false
	}
	p.state.Submitting = true
	p.state.Err = nil
	p.publish()
	accepted := p.submit.Submit(work, func(err error) {
		p.state.Submitting = false
		if err != nil {
			p.fail(err)
			return
		}
		p.state.Mode, p.state.Err = Browsing, nil
		p.publish()
		p.Refresh()
	})
	if !accepted {
		p.state.Submitting = false
		p.publish()
	}
	return accepted
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
func (p *Presenter) reselect() {
	if p.state.Selected == nil {
		if len(p.state.Items) > 0 {
			p.state.Selected = cloneDrink(p.state.Items[0])
		}
		return
	}
	id := p.state.Selected.ID
	for _, item := range p.state.Items {
		if item.ID == id {
			p.state.Selected = cloneDrink(item)
			return
		}
	}
	p.state.Selected = nil
}

func (p *Presenter) formDrink() (*models.Drink, error) {
	f := p.state.Form
	name := strings.TrimSpace(f.Name)
	if name == "" {
		return nil, apperrors.Invalidf("name is required")
	}
	if len([]rune(name)) > 100 {
		return nil, apperrors.Invalidf("name must be at most 100 characters")
	}
	description := strings.TrimSpace(f.Description)
	if len([]rune(description)) > 500 {
		return nil, apperrors.Invalidf("description must be at most 500 characters")
	}
	category := models.DrinkCategory(f.Category)
	if category == "" {
		return nil, apperrors.Invalidf("category is required")
	}
	if err := category.Validate(); err != nil {
		return nil, err
	}
	glass := models.GlassType(f.Glass)
	if glass == "" {
		return nil, apperrors.Invalidf("glass is required")
	}
	if err := glass.Validate(); err != nil {
		return nil, err
	}
	recipe, err := parseRecipe(f)
	if err != nil {
		return nil, err
	}
	d := &models.Drink{Name: name, Category: category, Glass: glass, Description: description, Recipe: recipe}
	if p.state.Mode == Editing && p.state.Selected != nil {
		d.ID = p.state.Selected.ID
	}
	return d, nil
}
func parseRecipe(f Form) (models.Recipe, error) {
	r := models.Recipe{Garnish: strings.TrimSpace(f.Garnish), Steps: nonEmptyLines(f.Steps)}
	for i, row := range f.Recipe {
		if row.Ingredient.IsZero() {
			return r, apperrors.Invalidf("recipe ingredient %d is required", i+1)
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(row.Amount), 64)
		if err != nil {
			return r, apperrors.Invalidf("recipe ingredient %d has invalid amount", i+1)
		}
		amount, err := measurement.NewAmount(value, row.Unit)
		if err != nil {
			return r, err
		}
		r.Ingredients = append(r.Ingredients, models.RecipeIngredient{IngredientID: row.Ingredient, Amount: amount, Optional: row.Optional, Substitutes: append([]entity.IngredientID(nil), row.Substitutes...)})
	}
	if err := r.Validate(); err != nil {
		return r, err
	}
	return r, nil
}
func nonEmptyLines(value string) []string {
	var out []string
	for _, v := range strings.Split(value, "\n") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
func formFromDrink(d *models.Drink) Form {
	f := Form{Name: d.Name, Category: string(d.Category), Glass: string(d.Glass), Description: d.Description, Garnish: d.Recipe.Garnish, Steps: strings.Join(d.Recipe.Steps, "\n"), Tags: d.Tags.Canonical().String(), ReplaceTags: true}
	for _, ingredient := range d.Recipe.Ingredients {
		f.Recipe = append(f.Recipe, RecipeRow{Ingredient: ingredient.IngredientID, Amount: strconv.FormatFloat(ingredient.Amount.Value(), 'f', -1, 64), Unit: ingredient.Amount.Unit(), Optional: ingredient.Optional, Substitutes: append([]entity.IngredientID(nil), ingredient.Substitutes...)})
	}
	f.Tags, _ = tag.FormatCollection(d.Tags)
	return f
}

func cloneState(in State) State {
	out := in
	out.Items = cloneDrinks(in.Items)
	out.Selected = cloneDrink(in.Selected)
	out.Ingredients = append([]IngredientOption(nil), in.Ingredients...)
	out.Form = cloneForm(in.Form)
	return out
}
func cloneForm(in Form) Form {
	out := in
	out.Recipe = append([]RecipeRow(nil), in.Recipe...)
	for i := range out.Recipe {
		out.Recipe[i].Substitutes = append([]entity.IngredientID(nil), in.Recipe[i].Substitutes...)
	}
	return out
}
func cloneDrinks(in []*models.Drink) []*models.Drink {
	out := make([]*models.Drink, len(in))
	for i, d := range in {
		out[i] = cloneDrink(d)
	}
	return out
}
func cloneDrink(d *models.Drink) *models.Drink {
	if d == nil {
		return nil
	}
	out := *d
	out.Tags = append(tag.Tags(nil), d.Tags...)
	out.Recipe.Ingredients = append([]models.RecipeIngredient(nil), d.Recipe.Ingredients...)
	for i := range out.Recipe.Ingredients {
		out.Recipe.Ingredients[i].Substitutes = append([]entity.IngredientID(nil), d.Recipe.Ingredients[i].Substitutes...)
	}
	out.Recipe.Steps = append([]string(nil), d.Recipe.Steps...)
	return &out
}
