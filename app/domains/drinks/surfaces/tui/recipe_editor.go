package tui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredients "github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	ingredientmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ingredientOption deliberately keeps the catalog's identity separate from its
// label. In particular, ingredient names are not encoded as comma-separated
// values when selecting multiple substitutes.
type ingredientOption struct {
	ID   entity.IngredientID
	Name string
}

type recipeRowEditor struct {
	ingredient      entity.IngredientID
	ingredientQuery textinput.Model
	amount          textinput.Model
	unit            measurement.Unit
	optional        bool
	substitutes     []entity.IngredientID
	substituteQuery textinput.Model
	candidate       int
}

type recipeControl uint8

const (
	controlIngredient recipeControl = iota
	controlAmount
	controlUnit
	controlOptional
	controlSubstitutes
	controlRemoveIngredient
	controlAddIngredient
	controlStep
	controlRemoveStep
	controlAddStep
	controlGarnish
)

type recipeFocus struct {
	kind  recipeControl
	index int
}

type ingredientCatalogLoadedMsg struct {
	owner   uint64
	options []ingredientOption
	err     error
}

var recipeEditorSequence atomic.Uint64

// RecipeEditor is the Drinks TUI's structured recipe field. It is bespoke to
// Bubble Tea: it owns terminal focus, searchable pickers and dynamic controls,
// while returning the same domain Recipe accepted by the application module.
type RecipeEditor struct {
	app      *app.Session
	owner    uint64
	styles   forms.FormStyles
	rows     []recipeRowEditor
	steps    []textinput.Model
	garnish  textinput.Model
	catalog  []ingredientOption
	loading  bool
	loadErr  error
	err      error
	focused  bool
	position int
	width    int
}

func NewRecipeEditor(session *app.Session, styles forms.FormStyles, initial models.Recipe) *RecipeEditor {
	e := &RecipeEditor{app: session, owner: recipeEditorSequence.Add(1), styles: styles, loading: true}
	e.garnish = newRecipeInput(initial.Garnish)
	for _, ingredient := range initial.Ingredients {
		e.rows = append(e.rows, newRecipeRow(ingredient))
	}
	if len(e.rows) == 0 {
		e.rows = append(e.rows, newRecipeRow(models.RecipeIngredient{}))
	}
	for _, step := range initial.Steps {
		e.steps = append(e.steps, newRecipeInput(step))
	}
	if len(e.steps) == 0 {
		e.steps = append(e.steps, newRecipeInput(""))
	}
	return e
}

func newRecipeRow(value models.RecipeIngredient) recipeRowEditor {
	unit := measurement.UnitOz
	amount := ""
	if value.Amount != nil {
		unit = value.Amount.Unit()
		amount = strconv.FormatFloat(value.Amount.Value(), 'f', -1, 64)
	}
	return recipeRowEditor{
		ingredient: value.IngredientID, amount: newRecipeInput(amount), unit: unit,
		optional: value.Optional, substitutes: append([]entity.IngredientID(nil), value.Substitutes...),
		ingredientQuery: newRecipeInput(""), substituteQuery: newRecipeInput(""),
	}
}

func newRecipeInput(value string) textinput.Model {
	in := textinput.New()
	in.Prompt = ""
	in.SetValue(value)
	return in
}

func (e *RecipeEditor) Init() tea.Cmd {
	e.loading = true
	owner := e.owner
	return func() tea.Msg {
		items, err := paging.Collect(func(cursor paging.Cursor) (paging.Page[*ingredientmodels.Ingredient], error) {
			return e.app.Ingredients.List(e.context(), ingredients.ListRequest{Cursor: cursor})
		})
		options := make([]ingredientOption, 0, len(items))
		for _, item := range items {
			if item != nil {
				options = append(options, ingredientOption{ID: item.ID, Name: item.Name})
			}
		}
		return ingredientCatalogLoadedMsg{owner: owner, options: options, err: err}
	}
}

// AcceptCatalog handles catalog completion even while another form field owns
// focus. The owner token prevents a late response from a discarded form from
// modifying a newly opened editor.
func (e *RecipeEditor) AcceptCatalog(msg ingredientCatalogLoadedMsg) bool {
	if msg.owner != e.owner {
		return false
	}
	e.loading, e.loadErr = false, msg.err
	if msg.err == nil {
		e.catalog = append([]ingredientOption(nil), msg.options...)
		e.resolveQueries()
	}
	return true
}

func (e *RecipeEditor) Update(msg tea.Msg) (forms.Field, tea.Cmd) {
	if loaded, ok := msg.(ingredientCatalogLoadedMsg); ok {
		e.AcceptCatalog(loaded)
		return e, nil
	}
	if !e.focused {
		return e, nil
	}
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return e, nil
	}
	controls := e.controls()
	if len(controls) == 0 {
		return e, nil
	}
	if e.position >= len(controls) {
		e.position = len(controls) - 1
	}
	focus := controls[e.position]
	switch keyMsg.String() {
	case "ctrl+n":
		e.moveFocus(1)
		return e, nil
	case "ctrl+p":
		e.moveFocus(-1)
		return e, nil
	}

	switch focus.kind {
	case controlIngredient:
		return e.updatePicker(keyMsg, focus.index, false)
	case controlAmount:
		row := &e.rows[focus.index]
		row.amount, _ = row.amount.Update(keyMsg)
	case controlUnit:
		if keyMsg.String() == "left" || keyMsg.String() == "up" {
			e.moveUnit(focus.index, -1)
		} else if keyMsg.String() == "right" || keyMsg.String() == "down" || keyMsg.String() == "enter" {
			e.moveUnit(focus.index, 1)
		}
	case controlOptional:
		if keyMsg.String() == "enter" || keyMsg.String() == " " {
			e.rows[focus.index].optional = !e.rows[focus.index].optional
		}
	case controlSubstitutes:
		return e.updatePicker(keyMsg, focus.index, true)
	case controlRemoveIngredient:
		if keyMsg.String() == "enter" && len(e.rows) > 1 {
			e.rows = slices.Delete(e.rows, focus.index, focus.index+1)
			e.clampFocus()
		}
	case controlAddIngredient:
		if keyMsg.String() == "enter" {
			e.rows = append(e.rows, newRecipeRow(models.RecipeIngredient{}))
			e.position = e.controlIndex(controlIngredient, len(e.rows)-1)
			e.syncFocus()
		}
	case controlStep:
		e.steps[focus.index], _ = e.steps[focus.index].Update(keyMsg)
	case controlRemoveStep:
		if keyMsg.String() == "enter" && len(e.steps) > 1 {
			e.steps = slices.Delete(e.steps, focus.index, focus.index+1)
			e.clampFocus()
		}
	case controlAddStep:
		if keyMsg.String() == "enter" {
			e.steps = append(e.steps, newRecipeInput(""))
			e.position = e.controlIndex(controlStep, len(e.steps)-1)
			e.syncFocus()
		}
	case controlGarnish:
		e.garnish, _ = e.garnish.Update(keyMsg)
	}
	return e, nil
}

func (e *RecipeEditor) updatePicker(msg tea.KeyMsg, rowIndex int, substitutes bool) (forms.Field, tea.Cmd) {
	row := &e.rows[rowIndex]
	query := &row.ingredientQuery
	if substitutes {
		query = &row.substituteQuery
	}
	candidates := e.matches(query.Value())
	switch msg.String() {
	case "up":
		row.candidate--
	case "down":
		row.candidate++
	case "enter":
		if len(candidates) == 0 {
			return e, nil
		}
		row.candidate = wrapped(row.candidate, len(candidates))
		selected := candidates[row.candidate]
		if substitutes {
			if selected.ID != row.ingredient {
				if at := slices.Index(row.substitutes, selected.ID); at >= 0 {
					row.substitutes = slices.Delete(row.substitutes, at, at+1)
					if len(row.substitutes) == 0 {
						row.substitutes = nil
					}
				} else {
					row.substitutes = append(row.substitutes, selected.ID)
				}
			}
		} else {
			row.ingredient = selected.ID
			row.substitutes = slices.DeleteFunc(row.substitutes, func(id entity.IngredientID) bool { return id == selected.ID })
			query.SetValue(selected.Name)
		}
		return e, nil
	default:
		previous := query.Value()
		*query, _ = query.Update(msg)
		if query.Value() != previous {
			row.candidate = 0
			if !substitutes {
				row.ingredient = entity.IngredientID{}
			}
		}
	}
	if len(candidates) > 0 {
		row.candidate = wrapped(row.candidate, len(candidates))
	}
	return e, nil
}

func (e *RecipeEditor) View() string {
	chunks, _ := e.renderChunks()
	return strings.Join(chunks, "\n")
}

func (e *RecipeEditor) renderChunks() ([]string, int) {
	var chunks []string
	focusLine := 0
	appendChunk := func(value string, focused bool) {
		if focused {
			focusLine = lineCount(strings.Join(chunks, "\n"))
		}
		chunks = append(chunks, value)
	}
	appendChunk(e.styles.LabelRequired.Render("Recipe *"), false)
	if e.loading {
		appendChunk(e.styles.Help.Render("Loading complete ingredient catalog…"), false)
	}
	if e.loadErr != nil {
		appendChunk(e.styles.Error.Render("Ingredient catalog: "+e.loadErr.Error()), false)
	}
	controls := e.controls()
	focus := recipeFocus{}
	if e.focused && len(controls) > 0 {
		focus = controls[min(e.position, len(controls)-1)]
	}
	for i, row := range e.rows {
		focused := focus.kind == controlIngredient && focus.index == i
		appendChunk(e.styles.Label.Render(fmt.Sprintf("Ingredient %d", i+1)), focused)
		appendChunk(e.renderInput("  Ingredient", e.ingredientText(row), focused), false)
		if focus.kind == controlIngredient && focus.index == i {
			matches, selected := e.renderMatches(row.ingredientQuery.Value(), row.candidate, nil)
			for index, match := range matches {
				appendChunk(match, index == selected)
			}
		}
		focused = focus.kind == controlAmount && focus.index == i
		appendChunk(e.renderInput("  Amount", row.amount.View(), focused), focused)
		focused = focus.kind == controlUnit && focus.index == i
		appendChunk(e.renderChoice("  Unit", string(row.unit), focused), focused)
		focused = focus.kind == controlOptional && focus.index == i
		appendChunk(e.renderChoice("  Optional", yesNo(row.optional), focused), focused)
		focused = focus.kind == controlSubstitutes && focus.index == i
		appendChunk(e.renderInput("  Substitutes", e.substituteText(row), focused), focused)
		if focus.kind == controlSubstitutes && focus.index == i {
			matches, selected := e.renderMatches(row.substituteQuery.Value(), row.candidate, row.substitutes)
			for index, match := range matches {
				appendChunk(match, index == selected)
			}
		}
		if len(e.rows) > 1 {
			focused = focus.kind == controlRemoveIngredient && focus.index == i
			appendChunk(e.renderChoice("  Remove ingredient", "press enter", focused), focused)
		}
	}
	focused := focus.kind == controlAddIngredient
	appendChunk(e.renderChoice("Add ingredient", "press enter", focused), focused)
	for i := range e.steps {
		focused = focus.kind == controlStep && focus.index == i
		appendChunk(e.renderInput(fmt.Sprintf("Step %d", i+1), e.steps[i].View(), focused), focused)
		if len(e.steps) > 1 {
			focused = focus.kind == controlRemoveStep && focus.index == i
			appendChunk(e.renderChoice("  Remove step", "press enter", focused), focused)
		}
	}
	focused = focus.kind == controlAddStep
	appendChunk(e.renderChoice("Add step", "press enter", focused), focused)
	focused = focus.kind == controlGarnish
	appendChunk(e.renderInput("Garnish", e.garnish.View(), focused), focused)
	if e.err != nil {
		appendChunk(e.styles.Error.Render(e.err.Error()), false)
	}
	return chunks, focusLine
}

func (e *RecipeEditor) focusLine() int { _, line := e.renderChunks(); return line }

func (e *RecipeEditor) renderInput(label, value string, focused bool) string {
	style := e.styles.Input
	if focused {
		style = e.styles.InputFocused
	}
	return e.styles.Label.Render(label) + "\n" + style.Render(value)
}

func (e *RecipeEditor) renderChoice(label, value string, focused bool) string {
	return e.renderInput(label, value, focused)
}

func (e *RecipeEditor) renderMatches(query string, selected int, checked []entity.IngredientID) ([]string, int) {
	matches := e.matches(query)
	if len(matches) == 0 {
		return []string{e.styles.Help.Render("    no matching ingredients")}, 0
	}
	selected = wrapped(selected, len(matches))
	start := 0
	if selected >= 5 {
		start = selected - 4
	}
	end := min(start+5, len(matches))
	out := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		option := matches[i]
		mark := "  "
		if slices.Contains(checked, option.ID) {
			mark = "✓ "
		}
		line := "    " + mark + option.Name
		if i == selected {
			line = e.styles.InputFocused.Render("> " + line)
		}
		out = append(out, line)
	}
	return out, selected - start
}

func (e *RecipeEditor) Focus()          { e.focused = true; e.syncFocus() }
func (e *RecipeEditor) Blur()           { e.focused = false; e.blurInputs() }
func (e *RecipeEditor) IsFocused() bool { return e.focused }
func (e *RecipeEditor) Label() string   { return "Recipe" }
func (e *RecipeEditor) Error() error    { return e.err }
func (e *RecipeEditor) SetWidth(width int) {
	e.width = width
	for i := range e.rows {
		e.rows[i].ingredientQuery.Width = width
		e.rows[i].amount.Width = width
		e.rows[i].substituteQuery.Width = width
	}
	for i := range e.steps {
		e.steps[i].Width = width
	}
	e.garnish.Width = width
}

func (e *RecipeEditor) Value() any {
	recipe, _ := e.recipe()
	return recipe
}

func (e *RecipeEditor) SetValue(value any) error {
	recipe, ok := value.(models.Recipe)
	if !ok {
		return fmt.Errorf("recipe value has type %T", value)
	}
	replacement := NewRecipeEditor(e.app, e.styles, recipe)
	replacement.owner, replacement.catalog, replacement.loading = e.owner, e.catalog, e.loading
	*e = *replacement
	e.resolveQueries()
	return nil
}

func (e *RecipeEditor) Validate() error {
	_, e.err = e.recipe()
	return e.err
}

func (e *RecipeEditor) recipe() (models.Recipe, error) {
	if e.loading {
		return models.Recipe{}, apperrors.Invalidf("ingredient catalog is still loading")
	}
	if e.loadErr != nil {
		return models.Recipe{}, e.loadErr
	}
	recipe := models.Recipe{Garnish: strings.TrimSpace(e.garnish.Value())}
	for i, row := range e.rows {
		if row.ingredient.IsZero() || !e.known(row.ingredient) {
			return recipe, apperrors.Invalidf("recipe ingredient %d must be selected from the catalog", i+1)
		}
		raw := strings.TrimSpace(row.amount.Value())
		if raw == "" {
			return recipe, apperrors.Invalidf("recipe ingredient %d amount is required", i+1)
		}
		if decimalPlaces(raw) > 6 {
			return recipe, apperrors.Invalidf("recipe ingredient %d amount has more than 6 decimal places", i+1)
		}
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return recipe, apperrors.Invalidf("recipe ingredient %d has invalid amount", i+1)
		}
		if value <= 0 {
			return recipe, apperrors.Invalidf("recipe ingredient %d amount must be positive", i+1)
		}
		if err := row.unit.Validate(); err != nil {
			return recipe, err
		}
		for _, substitute := range row.substitutes {
			if !e.known(substitute) {
				return recipe, apperrors.Invalidf("recipe ingredient %d contains an unknown substitute", i+1)
			}
			if substitute == row.ingredient {
				return recipe, apperrors.Invalidf("recipe ingredient %d cannot substitute itself", i+1)
			}
		}
		amount, err := measurement.NewAmount(value, row.unit)
		if err != nil {
			return recipe, err
		}
		recipe.Ingredients = append(recipe.Ingredients, models.RecipeIngredient{IngredientID: row.ingredient, Amount: amount, Optional: row.optional, Substitutes: append([]entity.IngredientID(nil), row.substitutes...)})
	}
	for _, input := range e.steps {
		if step := strings.TrimSpace(input.Value()); step != "" {
			recipe.Steps = append(recipe.Steps, step)
		}
	}
	if err := recipe.Validate(); err != nil {
		return recipe, err
	}
	return recipe, nil
}

func (e *RecipeEditor) controls() []recipeFocus {
	var out []recipeFocus
	for i := range e.rows {
		out = append(out, recipeFocus{controlIngredient, i}, recipeFocus{controlAmount, i}, recipeFocus{controlUnit, i}, recipeFocus{controlOptional, i}, recipeFocus{controlSubstitutes, i})
		if len(e.rows) > 1 {
			out = append(out, recipeFocus{controlRemoveIngredient, i})
		}
	}
	out = append(out, recipeFocus{kind: controlAddIngredient})
	for i := range e.steps {
		out = append(out, recipeFocus{controlStep, i})
		if len(e.steps) > 1 {
			out = append(out, recipeFocus{controlRemoveStep, i})
		}
	}
	out = append(out, recipeFocus{kind: controlAddStep}, recipeFocus{kind: controlGarnish})
	return out
}

func (e *RecipeEditor) moveFocus(delta int) {
	controls := e.controls()
	if len(controls) == 0 {
		return
	}
	e.position = wrapped(e.position+delta, len(controls))
	e.syncFocus()
}
func (e *RecipeEditor) clampFocus() {
	if e.position >= len(e.controls()) {
		e.position = len(e.controls()) - 1
	}
	e.syncFocus()
}
func (e *RecipeEditor) controlIndex(kind recipeControl, index int) int {
	for i, c := range e.controls() {
		if c.kind == kind && c.index == index {
			return i
		}
	}
	return 0
}
func (e *RecipeEditor) syncFocus() {
	e.blurInputs()
	if !e.focused || len(e.controls()) == 0 {
		return
	}
	f := e.controls()[min(e.position, len(e.controls())-1)]
	switch f.kind {
	case controlIngredient:
		e.rows[f.index].ingredientQuery.Focus()
	case controlAmount:
		e.rows[f.index].amount.Focus()
	case controlSubstitutes:
		e.rows[f.index].substituteQuery.Focus()
	case controlStep:
		e.steps[f.index].Focus()
	case controlGarnish:
		e.garnish.Focus()
	case controlUnit, controlOptional, controlRemoveIngredient, controlAddIngredient, controlRemoveStep, controlAddStep:
	}
}
func (e *RecipeEditor) blurInputs() {
	for i := range e.rows {
		e.rows[i].ingredientQuery.Blur()
		e.rows[i].amount.Blur()
		e.rows[i].substituteQuery.Blur()
	}
	for i := range e.steps {
		e.steps[i].Blur()
	}
	e.garnish.Blur()
}
func (e *RecipeEditor) moveUnit(row, delta int) {
	units := measurement.AllUnits()
	at := slices.Index(units, e.rows[row].unit)
	e.rows[row].unit = units[wrapped(at+delta, len(units))]
}
func (e *RecipeEditor) matches(query string) []ingredientOption {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return append([]ingredientOption(nil), e.catalog...)
	}
	var out []ingredientOption
	for _, option := range e.catalog {
		if strings.Contains(strings.ToLower(option.Name), query) {
			out = append(out, option)
		}
	}
	return out
}
func (e *RecipeEditor) known(id entity.IngredientID) bool {
	return slices.ContainsFunc(e.catalog, func(option ingredientOption) bool { return option.ID == id })
}
func (e *RecipeEditor) option(id entity.IngredientID) (ingredientOption, bool) {
	for _, option := range e.catalog {
		if option.ID == id {
			return option, true
		}
	}
	return ingredientOption{}, false
}
func (e *RecipeEditor) ingredientText(row recipeRowEditor) string {
	if row.ingredientQuery.Value() != "" {
		return row.ingredientQuery.View()
	}
	if option, ok := e.option(row.ingredient); ok {
		return option.Name
	}
	return "Search and press enter to select…"
}
func (e *RecipeEditor) selectedNames(ids []entity.IngredientID) string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if option, ok := e.option(id); ok {
			names = append(names, option.Name)
		}
	}
	if len(names) == 0 {
		return "None selected"
	}
	return strings.Join(names, " • ")
}
func (e *RecipeEditor) substituteText(row recipeRowEditor) string {
	selected := e.selectedNames(row.substitutes)
	if query := row.substituteQuery.View(); query != "" {
		return selected + "\nSearch: " + query
	}
	return selected + "\nSearch and press enter to toggle…"
}
func (e *RecipeEditor) resolveQueries() {
	for i := range e.rows {
		if option, ok := e.option(e.rows[i].ingredient); ok {
			e.rows[i].ingredientQuery.SetValue(option.Name)
		}
	}
}
func (e *RecipeEditor) context() *middleware.Context { return e.app.Context() }
func wrapped(value, length int) int {
	if length == 0 {
		return 0
	}
	value %= length
	if value < 0 {
		value += length
	}
	return value
}
func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
func decimalPlaces(value string) int {
	if at := strings.IndexByte(value, '.'); at >= 0 {
		return len(value) - at - 1
	}
	return 0
}
