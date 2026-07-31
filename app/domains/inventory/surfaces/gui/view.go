package gui

import (
	"fmt"
	"strconv"
	"time"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	toolkit "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

const (
	ControlFilter       = "inventory-filter"
	ControlThreshold    = "inventory-low-stock-threshold"
	ControlApplyFilter  = "inventory-apply-filter"
	ControlPrevious     = "inventory-previous"
	ControlNext         = "inventory-next"
	ControlRefresh      = "inventory-refresh"
	ControlAdjust       = "inventory-adjust"
	ControlSet          = "inventory-set"
	ControlTags         = "inventory-tags"
	ControlSelectPrefix = "inventory-select-"
	ControlAmount       = "inventory-form-amount"
	ControlQuantity     = "inventory-form-quantity"
	ControlCost         = "inventory-form-cost"
	ControlAdjustTags   = "inventory-adjust-tags"
	ControlSetTags      = "inventory-set-tags"
	ControlFormTags     = "inventory-form-tags"
	ControlSave         = "inventory-form-save"
	ControlCancel       = "inventory-form-cancel"
)

type View struct {
	presenter                                 *Presenter
	root                                      *framework.Container
	expression, threshold, amount, cost, tags *toolkit.SemanticEntry
	stock, limit, reason                      *widget.Select
	save                                      *toolkit.SemanticButton
	refresh, cancel                           *toolkit.SemanticButton
	rows                                      map[string]*toolkit.SemanticButton
}

var _ toolkit.View = (*View)(nil)
var _ toolkit.Activated = (*View)(nil)

func NewView(presenter *Presenter) *View {
	v := &View{presenter: presenter, root: container.NewStack()}
	presenter.OnChange(v.render)
	v.render(presenter.Snapshot())
	return v
}
func (v *View) Title() string                   { return "Inventory" }
func (v *View) Content() framework.CanvasObject { return v.root }
func (v *View) Activate()                       { v.presenter.Load() }
func (v *View) ExecuteCommand(command toolkit.Command) bool {
	state := v.presenter.Snapshot()
	switch command {
	case toolkit.CommandRefresh:
		return state.Mode == Browse && toolkit.Trigger(v.refresh)
	case toolkit.CommandSave:
		return state.Mode != Browse && toolkit.Trigger(v.save)
	case toolkit.CommandCancel:
		return state.Mode != Browse && toolkit.Trigger(v.cancel)
	case toolkit.CommandNew:
		return false
	}
	return false
}

func (v *View) render(state State) {
	v.expression = toolkit.NewEntry(ControlFilter)
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
	v.threshold = toolkit.NewEntry(ControlThreshold)
	v.threshold.SetPlaceHolder("Low-stock threshold")
	v.threshold.SetText(strconv.FormatFloat(state.LowStock, 'f', -1, 64))
	apply := toolkit.NewButton(ControlApplyFilter, "Apply", func() {
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
	prev := toolkit.NewButton(ControlPrevious, "Previous", v.presenter.PreviousPage)
	if len(state.History) == 0 {
		prev.Disable()
	}
	next := toolkit.NewButton(ControlNext, "Next", v.presenter.NextPage)
	if state.Next == "" {
		next.Disable()
	}
	refresh := toolkit.NewButton(ControlRefresh, "Refresh", v.presenter.Load)
	v.refresh = refresh
	adjust := toolkit.Primary(toolkit.NewButton(ControlAdjust, "Adjust stock", v.presenter.StartAdjust))
	set := toolkit.NewButton(ControlSet, "Set", v.presenter.StartSet)
	tags := toolkit.NewButton(ControlTags, "Tags", v.presenter.StartTags)
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
	rows := container.NewVBox()
	v.rows = make(map[string]*toolkit.SemanticButton, len(state.Rows))
	for i := range state.Rows {
		row := state.Rows[i]
		id := row.Inventory.ID
		button := toolkit.NewButton(ControlSelectPrefix+id.String(), fmt.Sprintf("%s  ·  %s  ·  %s  ·  %s", row.Ingredient.Name, row.Ingredient.Category, row.Quantity, row.Status), func() { v.presenter.Select(id) })
		if state.Mode != Browse || state.Submitting {
			button.Disable()
		}
		v.rows[id.String()] = button
		rows.Add(button)
	}
	if len(state.Rows) == 0 && state.Status == toolkit.Loaded {
		rows.Add(widget.NewLabel("No inventory found"))
	}
	status := ""
	switch state.Status {
	case toolkit.Loading:
		status = "Loading inventory…"
	case toolkit.Failed:
		status = "Unable to load inventory"
	case toolkit.Idle, toolkit.Loaded:
	}
	if state.Err != nil {
		status = "Error: " + state.Err.Error()
	}
	v.root.Objects = []framework.CanvasObject{toolkit.StandardListPage(toolkit.ListPage{
		Title: "Inventory", Subtitle: "Review stock levels and select an item to adjust or set its quantity.", Filters: filters,
		PrimaryActions: []framework.CanvasObject{adjust, refresh},
		OtherActions:   []framework.CanvasObject{set, tags},
		List:           container.NewScroll(rows), Detail: v.detail(state), Status: widget.NewLabel(status),
		Paging: container.NewHBox(prev, next), ListRatio: .42,
	})}
	v.root.Refresh()
}

func (v *View) detail(state State) framework.CanvasObject {
	if state.Mode != Browse {
		return v.form(state)
	}
	if state.Selected == nil {
		return toolkit.EmptyDetail("a stock item")
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
		v.amount = toolkit.NewEntry(ControlAmount)
		v.amount.SetPlaceHolder("Optional, e.g. +5.00 or -2.50")
		v.amount.SetText(state.Form.Amount)
		v.cost = toolkit.NewEntry(ControlCost)
		v.cost.SetPlaceHolder("Optional")
		v.cost.SetText(state.Form.Cost)
		v.reason = widget.NewSelect([]string{string(inventorymodels.ReasonReceived), string(inventorymodels.ReasonUsed), string(inventorymodels.ReasonSpilled), string(inventorymodels.ReasonExpired), string(inventorymodels.ReasonCorrected)}, nil)
		v.reason.SetSelected(string(state.Form.Reason))
		v.tags = toolkit.NewEntry(ControlAdjustTags)
		v.tags.SetText(state.Form.Tags)
		fields = widget.NewForm(widget.NewFormItem("Signed amount", v.amount), widget.NewFormItem("Cost per unit ("+currencyLabel+")", v.cost), widget.NewFormItem("Reason", v.reason), widget.NewFormItem("Tags (complete set)", v.tags))
	case Set:
		title = "Set Inventory"
		v.amount = toolkit.NewEntry(ControlQuantity)
		v.amount.SetText(state.Form.Amount)
		v.cost = toolkit.NewEntry(ControlCost)
		v.cost.SetPlaceHolder("Optional")
		v.cost.SetText(state.Form.Cost)
		v.tags = toolkit.NewEntry(ControlSetTags)
		v.tags.SetText(state.Form.Tags)
		fields = widget.NewForm(widget.NewFormItem("Quantity", v.amount), widget.NewFormItem("Cost per unit ("+currencyLabel+")", v.cost), widget.NewFormItem("Tags (complete set)", v.tags))
	case Tags:
		title = "Edit Inventory Tags"
		v.tags = toolkit.NewEntry(ControlFormTags)
		v.tags.SetPlaceHolder("featured, region=west")
		v.tags.SetText(state.Form.Tags)
		fields = container.NewVBox(widget.NewLabel("Complete tag set (CSV); clear to remove all tags"), v.tags)
	case Browse:
	}
	errorText := ""
	if state.Err != nil {
		errorText = "Error: " + state.Err.Error()
	}
	v.save = toolkit.NewButton(ControlSave, "Save", func() {
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
		case Browse:
		}
		v.presenter.Submit(form)
	})
	if state.Submitting {
		v.save.Disable()
	}
	cancel := toolkit.NewButton(ControlCancel, "Cancel", v.presenter.Cancel)
	v.cancel = cancel
	if state.Submitting {
		cancel.Disable()
	}
	return toolkit.StandardFormPage(toolkit.FormPage{Title: title, Fields: fields, Status: widget.NewLabel(errorText), Save: v.save, Cancel: cancel})
}
