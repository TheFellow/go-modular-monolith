package tui

import (
	"fmt"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsqueries "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/queries"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/charmbracelet/lipgloss"
)

// DetailViewModel renders a drink detail pane.
type DetailViewModel struct {
	styles  ListViewStyles
	width   int
	height  int
	drink   *models.Drink
	ctx     *middleware.Context
	queries *ingredientsqueries.Queries
}

func NewDetailViewModel(styles ListViewStyles, ctx *middleware.Context) *DetailViewModel {
	return &DetailViewModel{
		styles:  styles,
		ctx:     ctx,
		queries: ingredientsqueries.New(),
	}
}

func (d *DetailViewModel) SetSize(width, height int) {
	d.width = width
	d.height = height
}

func (d *DetailViewModel) SetDrink(drink *models.Drink) {
	d.drink = drink
}

func (d *DetailViewModel) View() string {
	if d.drink == nil {
		return d.styles.Subtitle.Render("Select a drink to view details")
	}

	lines := []string{
		d.styles.Title.Render(d.drink.Name),
		d.styles.Muted.Render("ID: " + d.drink.ID.String()),
		d.styles.Subtitle.Render("Category: ") + string(d.drink.Category),
		d.styles.Subtitle.Render("Glass: ") + string(d.drink.Glass),
	}

	if d.drink.Description != "" {
		lines = append(lines, "", d.styles.Subtitle.Render("Description"), d.drink.Description)
	}

	lines = append(lines, "", d.styles.Subtitle.Render("Recipe"))
	ingredientLines, err := d.renderIngredients(d.drink.Recipe.Ingredients)
	if err != nil {
		lines = append(lines, d.styles.ErrorText.Render(fmt.Sprintf("Error: %v", err)))
		content := strings.Join(lines, "\n")
		if d.width > 0 {
			content = lipgloss.NewStyle().Width(d.width).Render(content)
		}
		return content
	}
	lines = append(lines, ingredientLines...)

	if len(d.drink.Recipe.Steps) > 0 {
		lines = append(lines, "", d.styles.Subtitle.Render("Steps"))
		for i, step := range d.drink.Recipe.Steps {
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, step))
		}
	}

	if d.drink.Recipe.Garnish != "" {
		lines = append(lines, "", d.styles.Subtitle.Render("Garnish"), d.drink.Recipe.Garnish)
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
	ingredient, err := d.queries.Get(d.ctx, id)
	if err != nil {
		return "", errors.Internalf("could not load ingredient: %w", err)
	}

	if ingredient == nil {
		return "", errors.Internalf("ingredient %s missing", id.String())
	}

	name := strings.TrimSpace(ingredient.Name)
	if name == "" {
		return "", errors.Internalf("ingredient %s missing name", id.String())
	}
	return name, nil
}
