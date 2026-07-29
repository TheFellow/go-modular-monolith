package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	tuikeys "github.com/TheFellow/go-modular-monolith/main/tui/keys"
	tuistyles "github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type filterVM struct {
	form       *forms.Form
	category   *forms.SelectField
	expression *forms.TextField
	limit      *forms.NumberField
	err        error
}

func newFilterVM(req ingredients.ListRequest) *filterVM {
	options := []forms.SelectOption{{Label: "all", Value: models.Category("")}}
	for _, value := range models.AllCategories() {
		options = append(options, forms.SelectOption{Label: string(value), Value: value})
	}
	limit := req.Limit
	if limit <= 0 {
		limit = paging.DefaultLimit
	}
	v := &filterVM{
		category:   forms.NewSelectField("Category", options, forms.WithInitialValue(req.Category)),
		expression: forms.NewTextField("Expression", forms.WithInitialValue(req.Filter)),
		limit:      forms.NewNumberField("Page size", forms.WithRequired(), forms.WithMin(1), forms.WithInitialValue(limit)),
	}
	v.form = forms.New(tuistyles.App.Form, tuikeys.App.Form, v.category, v.expression, v.limit)
	return v
}
func (v *filterVM) Init() tea.Cmd { return v.form.Init() }
func (v *filterVM) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	v.form, cmd = v.form.Update(msg)
	return cmd
}
func (v *filterVM) View() string {
	out := "Filter Ingredients\n\n" + v.form.View() + "\n\nctrl+s apply • esc cancel"
	if v.err != nil {
		out = "Error: " + v.err.Error() + "\n\n" + out
	}
	return out
}
func (v *filterVM) Request() (ingredients.ListRequest, error) {
	if err := v.form.Validate(); err != nil {
		v.err = err
		return ingredients.ListRequest{}, err
	}
	limit, err := strconv.Atoi(fmt.Sprint(v.limit.Value()))
	if err != nil || limit <= 0 {
		v.err = fmt.Errorf("page size must be greater than zero")
		return ingredients.ListRequest{}, v.err
	}
	return ingredients.ListRequest{Category: v.category.Value().(models.Category), Filter: strings.TrimSpace(fmt.Sprint(v.expression.Value())), Limit: limit}, nil
}
func filterSubmit(msg tea.KeyMsg) bool { return key.Matches(msg, tuikeys.App.Form.Submit) }
