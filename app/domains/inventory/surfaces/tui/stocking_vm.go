package tui

import (
	"fmt"
	"strings"

	ingredientmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
	tea "github.com/charmbracelet/bubbletea"
)

type stockingLoadedMsg struct {
	Candidates []ingredientmodels.Ingredient
	Err        error
	Token      uint64
}
type stockingVM struct {
	candidates []ingredientmodels.Ingredient
	form       *forms.Form
	ingredient *forms.SelectField
	loading    bool
	err        error
}

func newStockingVM(candidates []ingredientmodels.Ingredient) *stockingVM {
	options := make([]forms.SelectOption, len(candidates))
	for i, candidate := range candidates {
		options[i] = forms.SelectOption{Label: candidate.Name + " (" + string(candidate.Unit) + ")", Value: candidate.ID}
	}
	field := forms.NewSelectField("Ingredient", options, forms.WithRequired())
	return &stockingVM{candidates: candidates, ingredient: field, form: forms.New(styles.Standard.Form, keys.Standard.Form, field)}
}
func (m *stockingVM) selected() (ingredientmodels.Ingredient, bool) {
	id, ok := m.ingredient.Value().(entity.IngredientID)
	if !ok {
		return ingredientmodels.Ingredient{}, false
	}
	for _, candidate := range m.candidates {
		if candidate.ID == id {
			return candidate, true
		}
	}
	return ingredientmodels.Ingredient{}, false
}
func (m *stockingVM) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return cmd
}
func (m *stockingVM) View() string {
	if m.loading {
		return "Loading ingredients…"
	}
	if m.err != nil {
		return fmt.Sprint("Error: ", m.err)
	}
	if len(m.candidates) == 0 {
		return "All active ingredients already have stock. Create an ingredient first."
	}
	return strings.Join([]string{"Receive new stock", "Choose an active ingredient without stock.", "", m.form.View()}, "\n")
}
