package tui

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/charmbracelet/lipgloss"
)

// DetailViewModel renders an ingredient detail pane.
type DetailViewModel struct {
	styles     tui.ListViewStyles
	width      int
	height     int
	ingredient optional.Value[models.Ingredient]
}

func NewDetailViewModel(styles tui.ListViewStyles) *DetailViewModel {
	return &DetailViewModel{styles: styles}
}

func (d *DetailViewModel) SetSize(width, height int) {
	d.width = width
	d.height = height
}

func (d *DetailViewModel) SetIngredient(ingredient optional.Value[models.Ingredient]) {
	d.ingredient = ingredient
}

func (d *DetailViewModel) View() string {
	ingredient, ok := d.ingredient.Unwrap()
	if !ok {
		return d.styles.Subtitle.Render("Select an ingredient to view details")
	}

	lines := []string{
		d.styles.Title.Render(ingredient.Name),
		d.styles.Muted.Render("ID: " + ingredient.ID.String()),
		d.styles.Subtitle.Render("Category: ") + string(ingredient.Category),
		d.styles.Subtitle.Render("Unit: ") + string(ingredient.Unit),
	}

	if strings.TrimSpace(ingredient.Description) != "" {
		lines = append(lines, "", d.styles.Subtitle.Render("Description"), ingredient.Description)
	}

	content := strings.Join(lines, "\n")
	if d.width > 0 {
		content = lipgloss.NewStyle().Width(d.width).Render(content)
	}
	return content
}
