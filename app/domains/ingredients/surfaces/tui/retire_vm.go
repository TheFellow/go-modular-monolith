package tui

import (
	"strconv"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// RetireIngredientVM captures an optional explicit permanent replacement.
// Leaving both fields blank performs a normal retirement and exposes the
// resulting degraded state for review.
type RetireIngredientVM struct {
	app                *app.Session
	ingredient         *models.Ingredient
	form               *forms.Form
	replacement, ratio *forms.TextField
	styles             forms.FormStyles
	keys               forms.FormKeys
	err                error
	submitting         bool
}

func NewRetireIngredientVM(application *app.Session, ingredient *models.Ingredient) *RetireIngredientVM {
	replacement := forms.NewTextField("Permanent replacement ingredient ID (optional)")
	ratio := forms.NewTextField("Replacement ratio (defaults to 1)", forms.WithInitialValue("1"))
	formStyles, formKeys := styles.Standard.Form, keys.Standard.Form
	return &RetireIngredientVM{app: application, ingredient: ingredient, form: forms.New(formStyles, formKeys, replacement, ratio), replacement: replacement, ratio: ratio, styles: formStyles, keys: formKeys}
}

func (m *RetireIngredientVM) Init() tea.Cmd      { return m.form.Init() }
func (m *RetireIngredientVM) SetWidth(width int) { m.form.SetWidth(width) }
func (m *RetireIngredientVM) IsEditing() bool    { return m.form.IsEditing() }

func (m *RetireIngredientVM) Update(msg tea.Msg) (*RetireIngredientVM, tea.Cmd) {
	if typed, ok := msg.(tea.KeyMsg); ok && key.Matches(typed, m.keys.Submit) {
		return m, m.submit()
	}
	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m *RetireIngredientVM) View() string {
	content := "Retire Ingredient\n\n" + m.form.View() + "\n\nLeave replacement blank to retire into review/degraded state."
	if m.err != nil {
		return m.styles.Error.Render("Error: "+m.err.Error()) + "\n\n" + content
	}
	return content
}

func (m *RetireIngredientVM) submit() tea.Cmd {
	if m.submitting || m.ingredient == nil {
		return nil
	}
	retirement := models.Retirement{}
	replacement := strings.TrimSpace(toString(m.replacement.Value()))
	if replacement != "" {
		id, err := entity.ParseIngredientID(replacement)
		if err != nil {
			m.err = err
			return nil
		}
		retirement.ReplacementID = id
		ratio := strings.TrimSpace(toString(m.ratio.Value()))
		if ratio != "" {
			value, err := strconv.ParseFloat(ratio, 64)
			if err != nil {
				m.err = errors.Invalidf("invalid replacement ratio %q", ratio)
				return nil
			}
			retirement.Ratio = value
		}
	}
	m.submitting = true
	return func() tea.Msg {
		retired, err := m.app.Ingredients.Retire(m.app.Context(), m.ingredient.ID, retirement)
		if err != nil {
			return DeleteErrorMsg{Err: err}
		}
		return IngredientDeletedMsg{Ingredient: retired}
	}
}
