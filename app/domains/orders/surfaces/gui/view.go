package gui

import (
	"fmt"
	"strconv"
	"strings"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	ui "github.com/TheFellow/go-modular-monolith/pkg/fyne"
)

type View struct {
	presenter                                                                *Presenter
	root                                                                     *framework.Container
	expression, menuQuery, drinkQuery, quantity, itemNotes, orderNotes, tags *ui.SemanticEntry
	status, limit, menus, drinks                                             *widget.Select
	rows                                                                     map[string]*ui.SemanticButton
	removeItems                                                              map[int]*ui.SemanticButton
	refresh, create, save, cancel                                            *ui.SemanticButton
}

var _ ui.View = (*View)(nil)
var _ ui.Activated = (*View)(nil)

func NewView(presenter *Presenter) *View {
	v := &View{presenter: presenter, root: container.NewStack()}
	presenter.Observe(v.render)
	return v
}
func (v *View) Title() string                   { return "Orders" }
func (v *View) Content() framework.CanvasObject { return v.root }
func (v *View) Activate()                       { v.presenter.Refresh() }
func (v *View) ExecuteCommand(command ui.Command) bool {
	state := v.presenter.State()
	switch command {
	case ui.CommandRefresh:
		return state.Mode == Browsing && ui.Trigger(v.refresh)
	case ui.CommandNew:
		return state.Mode == Browsing && ui.Trigger(v.create)
	case ui.CommandSave:
		return state.Mode != Browsing && ui.Trigger(v.save)
	case ui.CommandCancel:
		return state.Mode != Browsing && ui.Trigger(v.cancel)
	default:
		return false
	}
}

func (v *View) render(state State) {
	var body framework.CanvasObject
	if state.Mode == Browsing {
		body = v.browser(state)
	} else {
		body = v.form(state)
	}
	v.root.Objects = []framework.CanvasObject{body}
	v.root.Refresh()
}
func (v *View) browser(state State) framework.CanvasObject {
	v.expression = ui.NewEntry("orders-filter")
	v.expression.SetPlaceHolder(`status == "pending" && tags contains "featured"`)
	v.expression.SetText(state.Filter.Expression)
	v.status = widget.NewSelect([]string{"all", "pending", "completed", "cancelled"}, nil)
	selectedStatus := string(state.Filter.Status)
	if selectedStatus == "" {
		selectedStatus = "all"
	}
	v.status.SetSelected(selectedStatus)
	v.limit = widget.NewSelect([]string{"25", "50", "100"}, nil)
	v.limit.SetSelected(strconv.Itoa(state.Filter.Limit))
	if v.limit.Selected == "" {
		v.limit.SetSelected("100")
	}
	apply := ui.NewButton("orders-apply-filter", "Apply", func() {
		limit, _ := strconv.Atoi(v.limit.Selected)
		status := models.OrderStatus(v.status.Selected)
		if status == "all" {
			status = ""
		}
		v.presenter.ApplyFilter(Filter{Status: status, Expression: v.expression.Text, Limit: limit})
	})
	filters := container.NewBorder(nil, nil, container.NewHBox(v.status, v.limit), apply, v.expression)
	refresh := ui.NewButton("orders-refresh", "Refresh", v.presenter.Refresh)
	place := ui.NewButton("orders-place", "Place order", v.presenter.StartPlace)
	v.refresh, v.create = refresh, place
	complete := ui.NewButton("orders-complete", "Complete", v.presenter.ConfirmComplete)
	cancel := ui.NewButton("orders-cancel-order", "Cancel order", v.presenter.ConfirmCancel)
	tags := ui.NewButton("orders-tags", "Tags", v.presenter.StartTags)
	previous := ui.NewButton("orders-previous", "Previous", v.presenter.PreviousPage)
	if len(state.History) == 0 {
		previous.Disable()
	}
	next := ui.NewButton("orders-next", "Next", v.presenter.NextPage)
	if state.Next == "" {
		next.Disable()
	}
	mutable := state.Selected != nil && state.Selected.Order.Status == models.OrderStatusPending
	if !mutable {
		complete.Disable()
		cancel.Disable()
	}
	if state.Selected == nil {
		tags.Disable()
	}
	if state.Loading || state.Submitting || state.Confirming {
		for _, b := range []*ui.SemanticButton{refresh, place, complete, cancel, tags, previous, next, apply} {
			b.Disable()
		}
		v.expression.Disable()
		v.status.Disable()
		v.limit.Disable()
	}
	list := container.NewVBox()
	v.rows = make(map[string]*ui.SemanticButton, len(state.Rows))
	for i := range state.Rows {
		index := i
		row := state.Rows[i]
		button := ui.NewButton("orders-select-"+row.Order.ID.String(), fmt.Sprintf("%s  ·  %s  ·  %s", row.MenuName, row.Order.Status, formatTime(row.Order.CreatedAt)), func() { v.presenter.Select(index) })
		if state.Loading || state.Submitting || state.Confirming {
			button.Disable()
		}
		v.rows[row.Order.ID.String()] = button
		list.Add(button)
	}
	if len(state.Rows) == 0 && !state.Loading {
		list.Add(widget.NewLabel("No orders found"))
	}
	statusText := ""
	if state.Loading {
		statusText = "Loading orders…"
	}
	if state.Err != nil {
		statusText = "Error: " + state.Err.Error()
	}
	content := ui.ListDetail(container.NewScroll(list), v.detail(state), .42)
	return container.NewBorder(container.NewVBox(container.NewHBox(refresh, place, complete, cancel, tags, previous, next), filters, widget.NewLabel(statusText)), nil, nil, nil, content)
}
func (v *View) detail(state State) framework.CanvasObject {
	if state.Selected == nil {
		return container.NewPadded(widget.NewLabel("Select an order to view details"))
	}
	r := state.Selected
	tagText := r.Order.Tags.Canonical().String()
	if tagText == "" {
		tagText = "None"
	}
	fields := []framework.CanvasObject{widget.NewLabelWithStyle("Order", framework.TextAlignLeading, framework.TextStyle{Bold: true}), widget.NewLabel("ID: " + r.Order.ID.String()), widget.NewLabel("Menu: " + r.MenuName), widget.NewLabel("Status: " + string(r.Order.Status)), widget.NewLabel("Tags: " + tagText), widget.NewLabel("Created: " + formatTime(r.Order.CreatedAt))}
	if completed, ok := r.Order.CompletedAt.Unwrap(); ok {
		fields = append(fields, widget.NewLabel("Completed: "+formatTime(completed)))
	}
	if strings.TrimSpace(r.Order.Notes) != "" {
		fields = append(fields, widget.NewSeparator(), widget.NewLabelWithStyle("Notes", framework.TextAlignLeading, framework.TextStyle{Bold: true}), widget.NewLabel(r.Order.Notes))
	}
	fields = append(fields, widget.NewSeparator(), widget.NewLabelWithStyle("Items", framework.TextAlignLeading, framework.TextStyle{Bold: true}))
	for _, line := range r.Lines {
		label := fmt.Sprintf("%s  × %d  ·  %s", line.Name, line.Quantity, line.Total)
		if strings.TrimSpace(line.Notes) != "" {
			label += "\n" + line.Notes
		}
		fields = append(fields, widget.NewLabel(label))
	}
	fields = append(fields, widget.NewSeparator(), widget.NewLabelWithStyle("Total: "+r.Total, framework.TextAlignLeading, framework.TextStyle{Bold: true}))
	return container.NewPadded(container.NewVBox(fields...))
}
func (v *View) form(state State) framework.CanvasObject {
	if state.Mode == Tagging {
		return v.tagForm(state)
	}
	v.menuQuery = ui.NewEntry("orders-place-menu-search")
	v.menuQuery.SetPlaceHolder("Search published menus")
	v.menuQuery.SetText(state.Form.MenuQuery)
	searchMenus := ui.NewButton("orders-place-menu-search-apply", "Search", func() { v.presenter.SearchMenus(v.menuQuery.Text) })
	menuLabels := make([]string, len(state.Menus))
	menuIDs := make(map[string]entity.MenuID, len(state.Menus))
	for i, m := range state.Menus {
		label := m.Name + "  [" + m.ID.String() + "]"
		menuLabels[i] = label
		menuIDs[label] = m.ID
	}
	v.menus = widget.NewSelect(menuLabels, nil)
	for _, m := range state.Menus {
		if m.ID == state.Form.MenuID {
			v.menus.SetSelected(m.Name + "  [" + m.ID.String() + "]")
			break
		}
	}
	v.menus.OnChanged = func(value string) {
		if id, ok := menuIDs[value]; ok {
			v.presenter.ChooseMenu(id)
		}
	}
	v.drinkQuery = ui.NewEntry("orders-place-drink-search")
	v.drinkQuery.SetPlaceHolder("Search available drinks")
	v.drinkQuery.SetText(state.Form.DrinkQuery)
	searchDrinks := ui.NewButton("orders-place-drink-search-apply", "Search", func() { v.presenter.SearchDrinks(v.drinkQuery.Text) })
	drinkLabels := make([]string, len(state.Drinks))
	drinkIDs := make(map[string]entity.DrinkID, len(state.Drinks))
	for i, d := range state.Drinks {
		label := fmt.Sprintf("%s  ·  %s  ·  %s  [%s]", d.Name, d.Availability, d.Price, d.ID)
		drinkLabels[i] = label
		drinkIDs[label] = d.ID
	}
	v.drinks = widget.NewSelect(drinkLabels, nil)
	v.quantity = ui.NewEntry("orders-place-quantity")
	v.quantity.SetPlaceHolder("Quantity")
	v.itemNotes = ui.NewEntry("orders-place-item-notes")
	v.itemNotes.MultiLine = true
	v.itemNotes.SetPlaceHolder("Item notes (optional)")
	add := ui.NewButton("orders-place-add-item", "Add item", func() {
		qty, err := strconv.Atoi(strings.TrimSpace(v.quantity.Text))
		if err != nil {
			qty = 0
		}
		v.presenter.AddItem(drinkIDs[v.drinks.Selected], qty, v.itemNotes.Text)
	})
	items := container.NewVBox()
	v.removeItems = make(map[int]*ui.SemanticButton, len(state.Form.Items))
	for i, item := range state.Form.Items {
		index := i
		remove := ui.NewButton(fmt.Sprintf("orders-place-remove-%d", i), "Remove", func() { v.presenter.RemoveItem(index) })
		if state.Submitting || state.CatalogLoading {
			remove.Disable()
		}
		v.removeItems[i] = remove
		items.Add(container.NewBorder(nil, nil, nil, remove, widget.NewLabel(fmt.Sprintf("%s × %d%s", item.Name, item.Quantity, noteSuffix(item.Notes)))))
	}
	v.orderNotes = ui.NewEntry("orders-place-notes")
	v.orderNotes.MultiLine = true
	v.orderNotes.SetPlaceHolder("Order notes (optional)")
	v.orderNotes.SetText(state.Form.Notes)
	v.tags = ui.NewEntry("orders-place-tags")
	v.tags.SetText(state.Form.Tags)
	save := ui.NewButton("orders-place-save", "Place", func() {
		v.presenter.SetPlaceNotes(v.orderNotes.Text)
		v.presenter.SetPlaceTags(v.tags.Text)
		v.presenter.SavePlace()
	})
	back := ui.NewButton("orders-form-cancel", "Cancel", v.presenter.CancelForm)
	v.save, v.cancel = save, back
	if state.Submitting || state.CatalogLoading {
		for _, b := range []*ui.SemanticButton{searchMenus, searchDrinks, add, save, back} {
			b.Disable()
		}
		for _, e := range []*ui.SemanticEntry{v.menuQuery, v.drinkQuery, v.quantity, v.itemNotes, v.orderNotes} {
			e.Disable()
		}
		v.menus.Disable()
		v.drinks.Disable()
	}
	errorText := ""
	if state.CatalogLoading {
		errorText = "Loading published menus…"
	}
	if state.Err != nil {
		errorText = "Error: " + state.Err.Error()
	}
	return container.NewPadded(container.NewVBox(widget.NewLabelWithStyle("Place Order", framework.TextAlignLeading, framework.TextStyle{Bold: true}), widget.NewLabel(errorText), container.NewBorder(nil, nil, nil, searchMenus, v.menuQuery), widget.NewLabel("Published menu"), v.menus, container.NewBorder(nil, nil, nil, searchDrinks, v.drinkQuery), widget.NewLabel("Available drink"), v.drinks, widget.NewLabel("Quantity"), v.quantity, widget.NewLabel("Item notes"), v.itemNotes, add, widget.NewLabelWithStyle("Order items", framework.TextAlignLeading, framework.TextStyle{Bold: true}), items, widget.NewLabel("Order notes"), v.orderNotes, widget.NewLabel("Tags (complete set)"), v.tags, container.NewHBox(save, back)))
}
func (v *View) tagForm(state State) framework.CanvasObject {
	v.tags = ui.NewEntry("orders-tags-value")
	v.tags.SetPlaceHolder("featured, region=west")
	v.tags.SetText(state.Form.Tags)
	save := ui.NewButton("orders-tags-save", "Save", func() { v.presenter.SaveTags(v.tags.Text) })
	cancel := ui.NewButton("orders-form-cancel", "Cancel", v.presenter.CancelForm)
	v.save, v.cancel = save, cancel
	if state.Submitting {
		save.Disable()
		cancel.Disable()
		v.tags.Disable()
	}
	errorText := ""
	if state.Err != nil {
		errorText = "Error: " + state.Err.Error()
	}
	return container.NewPadded(container.NewVBox(widget.NewLabelWithStyle("Edit Order Tags", framework.TextAlignLeading, framework.TextStyle{Bold: true}), widget.NewLabel("Complete tag set (CSV); clear to remove all tags"), v.tags, widget.NewLabel(errorText), container.NewHBox(save, cancel)))
}
func noteSuffix(note string) string {
	if strings.TrimSpace(note) == "" {
		return ""
	}
	return " — " + note
}
