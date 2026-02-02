package tui

import (
	"fmt"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/queries"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/charmbracelet/lipgloss"
)

// DetailViewModel renders a drink detail pane.
type DetailViewModel struct {
	styles          tui.ListViewStyles
	width           int
	height          int
	drink           optional.Value[models.Drink]
	ctx             *middleware.Context
	queries         *queries.Queries
	ingredientNames map[entity.IngredientID]string
	ingredientErr   error
}

func NewDetailViewModel(styles tui.ListViewStyles, ctx *middleware.Context) *DetailViewModel {
	return &DetailViewModel{
		styles:  styles,
		ctx:     ctx,
		queries: queries.New(),
	}
}

func (d *DetailViewModel) SetSize(width, height int) {
	d.width = width
	d.height = height
}

func (d *DetailViewModel) SetDrink(drink optional.Value[models.Drink]) {
	d.drink = drink
	d.ingredientNames = nil
	d.ingredientErr = nil

	loaded, ok := drink.Unwrap()
	if !ok {
		return
	}

	ids := collectIngredientIDs(loaded.Recipe.Ingredients)
	if len(ids) == 0 {
		return
	}

	ingredients, err := d.queries.List(d.ctx, queries.ListFilter{IDs: ids})
	if err != nil {
		d.ingredientErr = err
		return
	}

	cache := make(map[entity.IngredientID]string, len(ingredients))
	for _, ingredient := range ingredients {
		if ingredient == nil {
			continue
		}
		name := strings.TrimSpace(ingredient.Name)
		if name != "" {
			cache[ingredient.ID] = name
		}
	}
	d.ingredientNames = cache
}

func (d *DetailViewModel) View() string {
	drink, ok := d.drink.Unwrap()
	if !ok {
		return d.styles.Subtitle.Render("Select a drink to view details")
	}

	lines := []string{
		d.styles.Title.Render(drink.Name),
		d.styles.Muted.Render("ID: " + drink.ID.String()),
		d.styles.Subtitle.Render("Category: ") + string(drink.Category),
		d.styles.Subtitle.Render("Glass: ") + string(drink.Glass),
	}

	if drink.Description != "" {
		lines = append(lines, "", d.styles.Subtitle.Render("Description"), drink.Description)
	}

	lines = append(lines, "", d.styles.Subtitle.Render("Recipe"))
	ingredientLines, err := d.renderIngredients(drink.Recipe.Ingredients)
	if err != nil {
		lines = append(lines, d.styles.ErrorText.Render(fmt.Sprintf("Error: %v", err)))
		content := strings.Join(lines, "\n")
		if d.width > 0 {
			content = lipgloss.NewStyle().Width(d.width).Render(content)
		}
		return content
	}
	lines = append(lines, ingredientLines...)

	if len(drink.Recipe.Steps) > 0 {
		lines = append(lines, "", d.styles.Subtitle.Render("Steps"))
		for i, step := range drink.Recipe.Steps {
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, step))
		}
	}

	if drink.Recipe.Garnish != "" {
		lines = append(lines, "", d.styles.Subtitle.Render("Garnish"), drink.Recipe.Garnish)
	}

	content := strings.Join(lines, "\n")
	if d.width > 0 {
		content = lipgloss.NewStyle().Width(d.width).Render(content)
	}
	return content
}

func (d *DetailViewModel) renderIngredients(items []models.RecipeIngredient) ([]string, error) {
	if len(items) == 0 {
		return []string{d.styles.Muted.Render("No ingredients")}, nil
	}

	lines := make([]string, 0, len(items))
	for _, item := range items {
		amount := item.Amount.String()
		optionalLabel := ""
		if item.Optional {
			optionalLabel = " (optional)"
		}
		name, err := d.ingredientName(item.IngredientID)
		if err != nil {
			return nil, errors.Internalf("could not load ingredient name: %w", err)
		}
		line := fmt.Sprintf("- %s %s%s", amount, name, optionalLabel)
		if len(item.Substitutes) > 0 {
			subs := make([]string, 0, len(item.Substitutes))
			for _, sub := range item.Substitutes {
				subName, err := d.ingredientName(sub)
				if err != nil {
					return nil, errors.Internalf("could not load ingredient name: %w", err)
				}
				subs = append(subs, subName)
			}
			line = fmt.Sprintf("%s [subs: %s]", line, strings.Join(subs, ", "))
		}
		lines = append(lines, line)
	}
	return lines, nil
}

func (d *DetailViewModel) ingredientName(id entity.IngredientID) (string, error) {
	if d.ingredientErr != nil {
		return "", errors.Internalf("could not load ingredients: %w", d.ingredientErr)
	}
	if d.ingredientNames == nil {
		return "", errors.Internalf("ingredient cache missing")
	}
	name, ok := d.ingredientNames[id]
	if !ok {
		return "", errors.Internalf("ingredient %s missing", id.String())
	}
	if name == "" {
		return "", errors.Internalf("ingredient %s missing name", id.String())
	}
	return name, nil
}

func collectIngredientIDs(items []models.RecipeIngredient) []entity.IngredientID {
	if len(items) == 0 {
		return nil
	}

	unique := make(map[entity.IngredientID]struct{}, len(items))
	for _, item := range items {
		if !item.IngredientID.IsZero() {
			unique[item.IngredientID] = struct{}{}
		}
		for _, sub := range item.Substitutes {
			if sub.IsZero() {
				continue
			}
			unique[sub] = struct{}{}
		}
	}

	ids := make([]entity.IngredientID, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}
	return ids
}
