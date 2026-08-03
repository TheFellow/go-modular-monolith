package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type filterVM struct {
	form             *forms.Form
	name, expression *forms.TextField
	category, glass  *forms.SelectField
	limit            *forms.NumberField
	err              error
}

func newFilterVM(req drinks.ListRequest) *filterVM {
	categories := []forms.SelectOption{{Label: "all", Value: models.DrinkCategory("")}}
	for _, value := range models.AllDrinkCategories() {
		categories = append(categories, forms.SelectOption{Label: string(value), Value: value})
	}
	glasses := []forms.SelectOption{{Label: "all", Value: models.GlassType("")}}
	for _, value := range models.AllGlassTypes() {
		glasses = append(glasses, forms.SelectOption{Label: string(value), Value: value})
	}
	limit := req.Limit
	if limit <= 0 {
		limit = paging.DefaultLimit
	}
	v := &filterVM{
		name:       forms.NewTextField("Exact name", forms.WithInitialValue(req.Name)),
		category:   forms.NewSelectField("Category", categories, forms.WithInitialValue(req.Category)),
		glass:      forms.NewSelectField("Glass", glasses, forms.WithInitialValue(req.Glass)),
		expression: forms.NewTextField("Expression", forms.WithInitialValue(req.Filter)),
		limit:      forms.NewNumberField("Page size", forms.WithRequired(), forms.WithMin(1), forms.WithInitialValue(limit)),
	}
	v.form = forms.New(styles.Standard.Form, keys.Standard.Form, v.name, v.category, v.glass, v.expression, v.limit)
	return v
}

func (v *filterVM) Init() tea.Cmd { return v.form.Init() }
func (v *filterVM) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	v.form, cmd = v.form.Update(msg)
	return cmd
}
func (v *filterVM) View() string {
	out := "Filter Drinks\n\n" + v.form.View() + "\n\nctrl+s apply • esc cancel"
	if v.err != nil {
		out = "Error: " + v.err.Error() + "\n\n" + out
	}
	return out
}
func (v *filterVM) Request() (drinks.ListRequest, error) {
	if err := v.form.Validate(); err != nil {
		v.err = err
		return drinks.ListRequest{}, err
	}
	limit, err := strconv.Atoi(fmt.Sprint(v.limit.Value()))
	if err != nil || limit <= 0 {
		v.err = fmt.Errorf("page size must be greater than zero")
		return drinks.ListRequest{}, v.err
	}
	return drinks.ListRequest{Name: strings.TrimSpace(fmt.Sprint(v.name.Value())), Category: v.category.Value().(models.DrinkCategory), Glass: v.glass.Value().(models.GlassType), Filter: strings.TrimSpace(fmt.Sprint(v.expression.Value())), Limit: limit}, nil
}
func filterSubmit(msg tea.KeyMsg) bool { return key.Matches(msg, keys.Standard.Form.Submit) }
