package tui

import (
	"fmt"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/charmbracelet/lipgloss"
)

// DetailViewModel renders a drink detail pane.
type DetailViewModel struct {
	styles      ListViewStyles
	width       int
	height      int
	drink       *models.Drink
	ingredients map[string]string
}

func NewDetailViewModel(styles ListViewStyles) *DetailViewModel {
	return &DetailViewModel{styles: styles}
}

func (d *DetailViewModel) SetSize(width, height int) {
	d.width = width
	d.height = height
}

func (d *DetailViewModel) SetDrink(drink *models.Drink) {
	d.drink = drink
}

func (d *DetailViewModel) SetIngredientNames(names map[string]string) {
	d.ingredients = names
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

	if strings.TrimSpace(d.drink.Description) != "" {
		lines = append(lines, "", d.styles.Subtitle.Render("Description"), d.drink.Description)
	}

	lines = append(lines, "", d.styles.Subtitle.Render("Recipe"))
	lines = append(lines, d.renderIngredients(d.drink.Recipe.Ingredients)...)

	if len(d.drink.Recipe.Steps) > 0 {
		lines = append(lines, "", d.styles.Subtitle.Render("Steps"))
		for i, step := range d.drink.Recipe.Steps {
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, step))
		}
	}

	if garnish := strings.TrimSpace(d.drink.Recipe.Garnish); garnish != "" {
		lines = append(lines, "", d.styles.Subtitle.Render("Garnish"), garnish)
	}

	content := strings.Join(lines, "\n")
	if d.width > 0 {
		content = lipgloss.NewStyle().Width(d.width).Render(content)
	}
	return content
}

func (d *DetailViewModel) renderIngredients(items []models.RecipeIngredient) []string {
	if len(items) == 0 {
		return []string{d.styles.Muted.Render("No ingredients")}
	}

	lines := make([]string, 0, len(items))
	for _, item := range items {
		amount := ""
		if item.Amount != nil {
			amount = item.Amount.String()
		}
		optionalLabel := ""
		if item.Optional {
			optionalLabel = " (optional)"
		}
		name := d.ingredientName(item.IngredientID.String())
		line := fmt.Sprintf("- %s %s%s", amount, name, optionalLabel)
		if len(item.Substitutes) > 0 {
			subs := make([]string, 0, len(item.Substitutes))
			for _, sub := range item.Substitutes {
				subs = append(subs, d.ingredientName(sub.String()))
			}
			line = fmt.Sprintf("%s [subs: %s]", line, strings.Join(subs, ", "))
		}
		lines = append(lines, line)
	}
	return lines
}

func (d *DetailViewModel) ingredientName(id string) string {
	if id == "" {
		return ""
	}
	if d.ingredients == nil {
		return id
	}
	if name, ok := d.ingredients[id]; ok && strings.TrimSpace(name) != "" {
		return name
	}
	return id
}
