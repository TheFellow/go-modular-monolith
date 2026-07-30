package tui

import (
	"fmt"
	"strconv"
	"strings"

	orders "github.com/TheFellow/go-modular-monolith/app/domains/orders"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	tuikeys "github.com/TheFellow/go-modular-monolith/main/tui/keys"
	tuistyles "github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	tea "github.com/charmbracelet/bubbletea"
)

type filterVM struct {
	form       *forms.Form
	status     *forms.SelectField
	expression *forms.TextField
	limit      *forms.NumberField
	err        error
}

func newFilterVM(req orders.ListRequest) *filterVM {
	limit := req.Limit
	if limit <= 0 {
		limit = paging.DefaultLimit
	}
	v := &filterVM{
		status: forms.NewSelectField("Status", []forms.SelectOption{
			{Label: "all", Value: models.OrderStatus("")},
			{Label: string(models.OrderStatusPending), Value: models.OrderStatusPending},
			{Label: string(models.OrderStatusCompleted), Value: models.OrderStatusCompleted},
			{Label: string(models.OrderStatusCancelled), Value: models.OrderStatusCancelled},
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
	out := "Filter Orders\n\n" + v.form.View() + "\n\nctrl+s apply • esc cancel"
	if v.err != nil {
		out = "Error: " + v.err.Error() + "\n\n" + out
	}
	return out
}
func (v *filterVM) Request() (orders.ListRequest, error) {
	if err := v.form.Validate(); err != nil {
		v.err = err
		return orders.ListRequest{}, err
	}
	limit, err := strconv.Atoi(fmt.Sprint(v.limit.Value()))
	if err != nil || limit <= 0 {
		v.err = fmt.Errorf("page size must be greater than zero")
		return orders.ListRequest{}, v.err
	}
	return orders.ListRequest{Status: v.status.Value().(models.OrderStatus), Filter: strings.TrimSpace(fmt.Sprint(v.expression.Value())), Limit: limit}, nil
}
