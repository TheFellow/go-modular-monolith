package fyne

import (
	"fmt"
	"strconv"
	"time"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	fyneui "github.com/TheFellow/go-modular-monolith/pkg/fyne"
)

type View struct {
	presenter                                 *Presenter
	root                                      *framework.Container
	expression, threshold, amount, cost, tags *fyneui.SemanticEntry
	stock, limit, reason                      *widget.Select
	save                                      *fyneui.SemanticButton
	refresh, cancel                           *fyneui.SemanticButton
	rows                                      map[string]*fyneui.SemanticButton
}

var _ fyneui.View = (*View)(nil)
var _ fyneui.Activated = (*View)(nil)

func NewView(presenter *Presenter) *View {
	v := &View{presenter: presenter, root: container.NewStack()}
	presenter.OnChange(v.render)
	v.render(presenter.Snapshot())
	return v
}
func (v *View) Title() string                   { return "Inventory" }
func (v *View) Content() framework.CanvasObject { return v.root }
func (v *View) Activate()                       { v.presenter.Load() }
func (v *View) ExecuteCommand(command fyneui.Command) bool {
	state := v.presenter.Snapshot()
	switch command {
	case fyneui.CommandRefresh:
		return state.Mode == Browse && fyneui.Trigger(v.refresh)
	case fyneui.CommandSave:
		return state.Mode != Browse && fyneui.Trigger(v.save)
	case fyneui.CommandCancel:
		return state.Mode != Browse && fyneui.Trigger(v.cancel)
	default:
		return false
	}
}

func (v *View) render(state State) {
	v.expression = fyneui.NewEntry("inventory-filter")
	v.expression.SetPlaceHolder(`quantity <= 5 && tags contains "featured"`)
	v.expression.SetText(state.Expression)
	v.stock = widget.NewSelect([]string{"all", "low stock"}, nil)
	if state.Stock == LowStock {
		v.stock.SetSelected("low stock")
	} else {
		v.stock.SetSelected("all")
	}
	v.limit = widget.NewSelect([]string{"25", "50", "100"}, nil)
	v.limit.SetSelected(strconv.Itoa(state.Limit))
	if v.limit.Selected == "" {
		v.limit.SetSelected("100")
	}
	v.threshold = fyneui.NewEntry("inventory-low-stock-threshold")
	v.threshold.SetPlaceHolder("Low-stock threshold")
	v.threshold.SetText(strconv.FormatFloat(state.LowStock, 'f', -1, 64))
	apply := fyneui.NewButton("inventory-apply-filter", "Apply", func() {
		limit, _ := strconv.Atoi(v.limit.Selected)
		threshold, err := strconv.ParseFloat(v.threshold.Text, 64)
		if err != nil {
			threshold = -1
		}
		stock := AllStock
		if v.stock.Selected == "low stock" {
			stock = LowStock
		}
		v.presenter.Filter(stock, v.expression.Text, threshold, limit)
	})
	filters := container.NewBorder(nil, nil, container.NewHBox(v.stock, v.threshold, v.limit), apply, v.expression)
	prev := fyneui.NewButton("inventory-previous", "Previous", v.presenter.PreviousPage)
	if len(state.History) == 0 {
		prev.Disable()
	}
	next := fyneui.NewButton("inventory-next", "Next", v.presenter.NextPage)
	if state.Next == "" {
		next.Disable()
	}
	refresh := fyneui.NewButton("inventory-refresh", "Refresh", v.presenter.Load)
	v.refresh = refresh
	adjust := fyneui.NewButton("inventory-adjust", "Adjust", v.presenter.StartAdjust)
	set := fyneui.NewButton("inventory-set", "Set", v.presenter.StartSet)
	tags := fyneui.NewButton("inventory-tags", "Tags", v.presenter.StartTags)
	if state.Mode != Browse || state.Submitting {
		refresh.Disable()
		adjust.Disable()
		set.Disable()
		tags.Disable()
		prev.Disable()
		next.Disable()
		apply.Disable()
		v.expression.Disable()
		v.stock.Disable()
		v.threshold.Disable()
		v.limit.Disable()
	}
	toolbar := container.NewHBox(refresh, adjust, set, tags, prev, next)
	rows := container.NewVBox()
	v.rows = make(map[string]*fyneui.SemanticButton, len(state.Rows))
	for i := range state.Rows {
		row := state.Rows[i]
		id := row.Inventory.ID
		button := fyneui.NewButton("inventory-select-"+id.String(), fmt.Sprintf("%s  ·  %s  ·  %s  ·  %s", row.Ingredient.Name, row.Ingredient.Category, row.Quantity, row.Status), func() { v.presenter.Select(id) })
		if state.Mode != Browse || state.Submitting {
			button.Disable()
		}
		v.rows[id.String()] = button
		rows.Add(button)
	}
	if len(state.Rows) == 0 && state.Status == fyneui.Loaded {
		rows.Add(widget.NewLabel("No inventory found"))
	}
	status := ""
	switch state.Status {
	case fyneui.Loading:
		status = "Loading inventory…"
	case fyneui.Failed:
		status = "Unable to load inventory"
	}
	if state.Err != nil {
		status = "Error: " + state.Err.Error()
	}
	content := fyneui.ListDetail(container.NewScroll(rows), v.detail(state), .42)
	v.root.Objects = []framework.CanvasObject{container.NewBorder(container.NewVBox(toolbar, filters, widget.NewLabel(status)), nil, nil, nil, content)}
	v.root.Refresh()
}

func (v *View) detail(state State) framework.CanvasObject {
	if state.Mode != Browse {
		return v.form(state)
	}
	if state.Selected == nil {
		return container.NewPadded(widget.NewLabel("Select a stock item to view details"))
	}
	r := state.Selected
	tags := r.Inventory.Tags.Canonical().String()
	if tags == "" {
		tags = "None"
	}
	labels := inventoryDetailLabels(r, tags)
	objects := []framework.CanvasObject{widget.NewLabelWithStyle(labels[0], framework.TextAlignLeading, framework.TextStyle{Bold: true})}
	for _, label := range labels[1:6] {
		objects = append(objects, widget.NewLabel(label))
	}
	objects = append(objects, widget.NewSeparator())
	for _, label := range labels[6:] {
		objects = append(objects, widget.NewLabel(label))
	}
	return container.NewPadded(container.NewVBox(objects...))
}

func inventoryDetailLabels(r *Row, tags string) []string {
	return []string{r.Ingredient.Name, "Ingredient ID: " + r.Ingredient.ID.String(), "Inventory ID: " + r.Inventory.ID.String(), "Category: " + string(r.Ingredient.Category), "Unit: " + string(r.Ingredient.Unit), "Tags: " + tags, "Quantity: " + r.Quantity, "Cost per unit: " + r.Cost, "Status: " + r.Status, "Last updated: " + formatInventoryTime(r.Inventory.LastUpdated)}
}

func formatInventoryTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}

func (v *View) form(state State) framework.CanvasObject {
	title := "Adjust Inventory"
	currencyLabel := "USD"
	if state.Selected != nil {
		if price, ok := state.Selected.Inventory.CostPerUnit.Unwrap(); ok {
			currencyLabel = price.Currency.String()
		}
	}
	var fields framework.CanvasObject
	switch state.Mode {
	case Adjust:
		v.amount = fyneui.NewEntry("inventory-form-amount")
		v.amount.SetPlaceHolder("Optional, e.g. +5.00 or -2.50")
		v.amount.SetText(state.Form.Amount)
		v.cost = fyneui.NewEntry("inventory-form-cost")
		v.cost.SetPlaceHolder("Optional")
		v.cost.SetText(state.Form.Cost)
		v.reason = widget.NewSelect([]string{string(inventorymodels.ReasonReceived), string(inventorymodels.ReasonUsed), string(inventorymodels.ReasonSpilled), string(inventorymodels.ReasonExpired), string(inventorymodels.ReasonCorrected)}, nil)
		v.reason.SetSelected(string(state.Form.Reason))
		v.tags = fyneui.NewEntry("inventory-adjust-tags")
		v.tags.SetText(state.Form.Tags)
		fields = widget.NewForm(widget.NewFormItem("Signed amount", v.amount), widget.NewFormItem("Cost per unit ("+currencyLabel+")", v.cost), widget.NewFormItem("Reason", v.reason), widget.NewFormItem("Tags (complete set)", v.tags))
	case Set:
		title = "Set Inventory"
		v.amount = fyneui.NewEntry("inventory-form-quantity")
		v.amount.SetText(state.Form.Amount)
		v.cost = fyneui.NewEntry("inventory-form-cost")
		v.cost.SetPlaceHolder("Optional")
		v.cost.SetText(state.Form.Cost)
		v.tags = fyneui.NewEntry("inventory-set-tags")
		v.tags.SetText(state.Form.Tags)
		fields = widget.NewForm(widget.NewFormItem("Quantity", v.amount), widget.NewFormItem("Cost per unit ("+currencyLabel+")", v.cost), widget.NewFormItem("Tags (complete set)", v.tags))
	case Tags:
		title = "Edit Inventory Tags"
		v.tags = fyneui.NewEntry("inventory-form-tags")
		v.tags.SetPlaceHolder("featured, region=west")
		v.tags.SetText(state.Form.Tags)
		fields = container.NewVBox(widget.NewLabel("Complete tag set (CSV); clear to remove all tags"), v.tags)
	}
	errorText := ""
	if state.Err != nil {
		errorText = "Error: " + state.Err.Error()
	}
	v.save = fyneui.NewButton("inventory-form-save", "Save", func() {
		form := Form{}
		switch state.Mode {
		case Adjust:
			form.Amount = v.amount.Text
			form.Cost = v.cost.Text
			form.Reason = inventorymodels.AdjustmentReason(v.reason.Selected)
			form.Tags, form.ReplaceTags = v.tags.Text, true
		case Set:
			form.Amount = v.amount.Text
			form.Cost = v.cost.Text
			form.Tags, form.ReplaceTags = v.tags.Text, true
		case Tags:
			form.Tags = v.tags.Text
		}
		v.presenter.Submit(form)
	})
	if state.Submitting {
		v.save.Disable()
	}
	cancel := fyneui.NewButton("inventory-form-cancel", "Cancel", v.presenter.Cancel)
	v.cancel = cancel
	if state.Submitting {
		cancel.Disable()
	}
	return container.NewPadded(container.NewVBox(widget.NewLabelWithStyle(title, framework.TextAlignLeading, framework.TextStyle{Bold: true}), widget.NewLabel(errorText), fields, container.NewHBox(v.save, cancel)))
}
