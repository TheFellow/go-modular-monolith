package tui

import (
	"fmt"
	"strconv"
	"strings"

	menus "github.com/TheFellow/go-modular-monolith/app/domains/menus"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	tuikeys "github.com/TheFellow/go-modular-monolith/main/tui/keys"
	tuistyles "github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	tea "github.com/charmbracelet/bubbletea"
)

type filterVM struct {
	form       *forms.Form
	status     *forms.SelectField
	expression *forms.TextField
	limit      *forms.NumberField
	err        error
}

func newFilterVM(req menus.ListRequest) *filterVM {
	limit := req.Limit
	if limit <= 0 {
		limit = paging.DefaultLimit
	}
	v := &filterVM{
		status: forms.NewSelectField("Status", []forms.SelectOption{
			{Label: "all", Value: models.MenuStatus("")},
			{Label: string(models.MenuStatusDraft), Value: models.MenuStatusDraft},
			{Label: string(models.MenuStatusPublished), Value: models.MenuStatusPublished},
			{Label: string(models.MenuStatusArchived), Value: models.MenuStatusArchived},
		}, forms.WithInitialValue(req.Status)),
		expression: forms.NewTextField("Expression", forms.WithInitialValue(req.Filter)),
		limit:      forms.NewNumberField("Page size", forms.WithRequired(), forms.WithMin(1), forms.WithInitialValue(limit)),
	}
	v.form = forms.New(tuistyles.App.Form, tuikeys.App.Form, v.status, v.expression, v.limit)
	return v
}

func (v *filterVM) Init() tea.Cmd { return v.form.Init() }
func (v *filterVM) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	v.form, cmd = v.form.Update(msg)
	return cmd
}
func (v *filterVM) View() string {
	out := "Filter Menus\n\n" + v.form.View() + "\n\nctrl+s apply • esc cancel"
	if v.err != nil {
		out = "Error: " + v.err.Error() + "\n\n" + out
	}
	return out
}
func (v *filterVM) Request() (menus.ListRequest, error) {
	if err := v.form.Validate(); err != nil {
		v.err = err
		return menus.ListRequest{}, err
	}
	limit, err := strconv.Atoi(fmt.Sprint(v.limit.Value()))
	if err != nil || limit <= 0 {
		v.err = fmt.Errorf("page size must be greater than zero")
		return menus.ListRequest{}, v.err
	}
	return menus.ListRequest{
		Status: v.status.Value().(models.MenuStatus), Filter: strings.TrimSpace(fmt.Sprint(v.expression.Value())), Limit: limit,
	}, nil
}
