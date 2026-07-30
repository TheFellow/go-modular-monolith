package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/surfaces/tui/keys"
	"github.com/TheFellow/go-modular-monolith/app/surfaces/tui/styles"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type filterVM struct {
	form       *forms.Form
	stock      *forms.SelectField
	expression *forms.TextField
	threshold  *forms.NumberField
	limit      *forms.NumberField
	err        error
}

func filterSubmit(msg tea.KeyMsg) bool { return key.Matches(msg, keys.App.Form.Submit) }

func newFilterVM(req inventory.ListRequest) *filterVM {
	threshold := inventory.DefaultLowStockThreshold
	stock := "all"
	if value, ok := req.LowStock.Unwrap(); ok {
		threshold = value
		stock = "low stock"
	}
	limit := req.Limit
	if limit <= 0 {
		limit = paging.DefaultLimit
	}
	v := &filterVM{
		stock:      forms.NewSelectField("Stock", []forms.SelectOption{{Label: "all", Value: "all"}, {Label: "low stock", Value: "low stock"}}, forms.WithInitialValue(stock)),
		expression: forms.NewTextField("Expression", forms.WithInitialValue(req.Filter)),
		threshold:  forms.NewNumberField("Low-stock threshold", forms.WithMin(0), forms.WithInitialValue(threshold)),
		limit:      forms.NewNumberField("Page size", forms.WithRequired(), forms.WithMin(1), forms.WithInitialValue(limit)),
	}
	v.form = forms.New(styles.App.Form, keys.App.Form, v.stock, v.expression, v.threshold, v.limit)
	return v
}

func (v *filterVM) Init() tea.Cmd { return v.form.Init() }
func (v *filterVM) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	v.form, cmd = v.form.Update(msg)
	return cmd
}
func (v *filterVM) View() string {
	out := "Filter Inventory\n\n" + v.form.View() + "\n\nctrl+s apply • esc cancel"
	if v.err != nil {
		out = "Error: " + v.err.Error() + "\n\n" + out
	}
	return out
}
func (v *filterVM) Request() (inventory.ListRequest, error) {
	if err := v.form.Validate(); err != nil {
		v.err = err
		return inventory.ListRequest{}, err
	}
	limit, err := strconv.Atoi(fmt.Sprint(v.limit.Value()))
	if err != nil || limit <= 0 {
		v.err = fmt.Errorf("page size must be greater than zero")
		return inventory.ListRequest{}, v.err
	}
	threshold, err := strconv.ParseFloat(fmt.Sprint(v.threshold.Value()), 64)
	if err != nil || threshold < 0 {
		v.err = fmt.Errorf("low-stock threshold must be zero or greater")
		return inventory.ListRequest{}, v.err
	}
	lowStock := optional.None[float64]()
	if fmt.Sprint(v.stock.Value()) == "low stock" {
		lowStock = optional.Some(threshold)
	}
	return inventory.ListRequest{Filter: strings.TrimSpace(fmt.Sprint(v.expression.Value())), LowStock: lowStock, Limit: limit}, nil
}
