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
	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

const (
	ControlFilter           = "orders-filter"
	ControlApplyFilter      = "orders-apply-filter"
	ControlRefresh          = "orders-refresh"
	ControlPlace            = "orders-place"
	ControlComplete         = "orders-complete"
	ControlCancelOrder      = "orders-cancel-order"
	ControlTags             = "orders-tags"
	ControlPrevious         = "orders-previous"
	ControlNext             = "orders-next"
	ControlSelectPrefix     = "orders-select-"
	ControlMenuSearch       = "orders-place-menu-search"
	ControlMenuSearchApply  = "orders-place-menu-search-apply"
	ControlDrinkSearch      = "orders-place-drink-search"
	ControlDrinkSearchApply = "orders-place-drink-search-apply"
	ControlQuantity         = "orders-place-quantity"
	ControlItemNotes        = "orders-place-item-notes"
	ControlAddItem          = "orders-place-add-item"
	ControlOrderNotes       = "orders-place-notes"
	ControlPlaceTags        = "orders-place-tags"
	ControlPlaceSave        = "orders-place-save"
	ControlFormCancel       = "orders-form-cancel"
	ControlTagValues        = "orders-tags-value"
	ControlTagSave          = "orders-tags-save"
)

type View struct {
	presenter                                                                *Presenter
	root                                                                     *framework.Container
	expression, menuQuery, drinkQuery, quantity, itemNotes, orderNotes, tags *ui.SemanticEntry
	limit, menus, drinks                                                     *widget.Select
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
func (v *View) HasUnsavedChanges() bool         { return v.presenter.State().Mode != Browsing }
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
	}
	return false
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
	v.expression = ui.NewEntry(ControlFilter)
	v.expression.SetPlaceHolder(`status == "pending" && tags contains "featured"`)
	v.expression.SetText(state.Filter.Expression)
	v.limit = widget.NewSelect([]string{"25", "50", "100"}, nil)
	v.limit.SetSelected(strconv.Itoa(state.Filter.Limit))
	if v.limit.Selected == "" {
		v.limit.SetSelected("100")
	}
	bar := ui.NewFilterBar(ControlFilter, ControlApplyFilter, `Filter orders (for example: tags contains "featured")`, state.Filter.Expression,
		[]ui.FilterPreset{{ID: "orders-status", Placeholder: "Status", Options: []ui.FilterOption{{Label: "Any status"}, {Label: "Pending", Expression: `status == "pending"`}, {Label: "Completed", Expression: `status == "completed"`}, {Label: "Cancelled", Expression: `status == "cancelled"`}}}}, nil,
		container.NewBorder(nil, nil, widget.NewLabel("Page size"), nil, v.limit), func(expression string) {
			limit, _ := strconv.Atoi(v.limit.Selected)
			v.presenter.ApplyFilter(Filter{Expression: expression, Limit: limit})
		})
	v.expression = bar.Expression
	apply := bar.Apply
	filters := bar.Content
	refresh := ui.NewButton(ControlRefresh, "Refresh", v.presenter.Refresh)
	place := ui.Primary(ui.NewButton(ControlPlace, "Place order", v.presenter.StartPlace))
	v.refresh, v.create = refresh, place
	complete := ui.NewButton(ControlComplete, "Complete", v.presenter.ConfirmComplete)
	cancel := ui.Destructive(ui.NewButton(ControlCancelOrder, "Cancel order", v.presenter.ConfirmCancel))
	tags := ui.NewButton(ControlTags, "Tags", v.presenter.StartTags)
	previous := ui.NewButton(ControlPrevious, "Previous", v.presenter.PreviousPage)
	if len(state.History) == 0 {
		previous.Disable()
	}
	next := ui.NewButton(ControlNext, "Next", v.presenter.NextPage)
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
		v.limit.Disable()
		bar.SetEnabled(false)
	}
	list := container.NewVBox()
	v.rows = make(map[string]*ui.SemanticButton, len(state.Rows))
	for i := range state.Rows {
		index := i
		row := state.Rows[i]
		button := ui.NewButton(ControlSelectPrefix+row.Order.ID.String(), fmt.Sprintf("%s  ·  %s  ·  %s", row.MenuName, row.Order.Status, formatTime(row.Order.CreatedAt)), func() { v.presenter.Select(index) })
		if state.Loading || state.Submitting || state.Confirming {
			button.Disable()
		}
		v.rows[row.Order.ID.String()] = button
		list.Add(button)
	}
	if len(state.Rows) == 0 && !state.Loading {
		list.Add(ui.EmptyCollection("orders", "Adjust the filters or place the first order."))
	}
	statusText := ""
	if state.Loading {
		statusText = "Loading orders…"
	}
	if state.Err != nil {
		statusText = "Error: " + state.Err.Error()
	}
	detailActions := []framework.CanvasObject(nil)
	if state.Mode == Browsing && state.Selected != nil {
		detailActions = []framework.CanvasObject{complete, tags, cancel}
	}
	return ui.StandardListPage(ui.ListPage{
		FilterDisclosure: bar.Advanced,
		Title:            "Orders", Subtitle: "Browse orders and select one to review its items and lifecycle actions.", Filters: filters,
		CollectionActions: []framework.CanvasObject{place, refresh},
		DetailActions:     detailActions,
		List:              container.NewScroll(list), Detail: v.detail(state), Status: widget.NewLabel(statusText),
		Paging: container.NewHBox(previous, next), ListRatio: .42,
	})
}
func (v *View) detail(state State) framework.CanvasObject {
	if state.Selected == nil {
		return ui.EmptyDetail("an order")
	}
	r := state.Selected
	fields := []framework.CanvasObject{widget.NewLabelWithStyle("Order", framework.TextAlignLeading, framework.TextStyle{Bold: true}), widget.NewLabel("ID: " + r.Order.ID.String()), widget.NewLabel("Menu: " + r.MenuName), widget.NewLabel("Status: " + string(r.Order.Status)), widget.NewLabelWithStyle("Tags", framework.TextAlignLeading, framework.TextStyle{Bold: true}), ui.TagPills([]string(r.Order.Tags.Canonical())), widget.NewLabel("Created: " + formatTime(r.Order.CreatedAt))}
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
	v.menuQuery = ui.NewEntry(ControlMenuSearch)
	v.menuQuery.SetPlaceHolder("Search published menus")
	v.menuQuery.SetText(state.Form.MenuQuery)
	searchMenus := ui.NewButton(ControlMenuSearchApply, "Search", func() { v.presenter.SearchMenus(v.menuQuery.Text) })
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
	v.drinkQuery = ui.NewEntry(ControlDrinkSearch)
	v.drinkQuery.SetPlaceHolder("Search available drinks")
	v.drinkQuery.SetText(state.Form.DrinkQuery)
	searchDrinks := ui.NewButton(ControlDrinkSearchApply, "Search", func() { v.presenter.SearchDrinks(v.drinkQuery.Text) })
	drinkLabels := make([]string, len(state.Drinks))
	drinkIDs := make(map[string]entity.DrinkID, len(state.Drinks))
	for i, d := range state.Drinks {
		label := fmt.Sprintf("%s  ·  %s  ·  %s  [%s]", d.Name, d.Availability, d.Price, d.ID)
		drinkLabels[i] = label
		drinkIDs[label] = d.ID
	}
	v.drinks = widget.NewSelect(drinkLabels, nil)
	v.quantity = ui.NewEntry(ControlQuantity)
	v.quantity.SetPlaceHolder("Quantity")
	v.itemNotes = ui.NewEntry(ControlItemNotes)
	v.itemNotes.MultiLine = true
	v.itemNotes.SetPlaceHolder("Item notes (optional)")
	add := ui.NewButton(ControlAddItem, "Add item", func() {
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
	v.orderNotes = ui.NewEntry(ControlOrderNotes)
	v.orderNotes.MultiLine = true
	v.orderNotes.SetPlaceHolder("Order notes (optional)")
	v.orderNotes.SetText(state.Form.Notes)
	v.tags = ui.NewEntry(ControlPlaceTags)
	v.tags.SetText(state.Form.Tags)
	save := ui.NewButton(ControlPlaceSave, "Place", func() {
		v.presenter.SetPlaceNotes(v.orderNotes.Text)
		v.presenter.SetPlaceTags(v.tags.Text)
		v.presenter.SavePlace()
	})
	back := ui.NewButton(ControlFormCancel, "Cancel", v.presenter.CancelForm)
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
	fields := container.NewVBox(container.NewBorder(nil, nil, nil, searchMenus, v.menuQuery), widget.NewLabel("Published menu"), v.menus, container.NewBorder(nil, nil, nil, searchDrinks, v.drinkQuery), widget.NewLabel("Available drink"), v.drinks, widget.NewLabel("Quantity"), v.quantity, widget.NewLabel("Item notes"), v.itemNotes, add, widget.NewLabelWithStyle("Order items", framework.TextAlignLeading, framework.TextStyle{Bold: true}), items, widget.NewLabel("Order notes"), v.orderNotes, widget.NewLabel("Tags (complete set)"), v.tags)
	return ui.StandardFormPage(ui.FormPage{Title: "Place Order", Subtitle: "Choose a published menu, add drinks, then place the order.", Fields: fields, Status: widget.NewLabel(errorText), Save: save, Cancel: back})
}
func (v *View) tagForm(state State) framework.CanvasObject {
	v.tags = ui.NewEntry(ControlTagValues)
	v.tags.SetPlaceHolder("featured, region=west")
	v.tags.SetText(state.Form.Tags)
	save := ui.NewButton(ControlTagSave, "Save", func() { v.presenter.SaveTags(v.tags.Text) })
	cancel := ui.NewButton(ControlFormCancel, "Cancel", v.presenter.CancelForm)
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
	return ui.StandardFormPage(ui.FormPage{Title: "Edit Order Tags", Subtitle: "Complete tag set (CSV); clear to remove all tags.", Fields: v.tags, Status: widget.NewLabel(errorText), Save: save, Cancel: cancel})
}
func noteSuffix(note string) string {
	if strings.TrimSpace(note) == "" {
		return ""
	}
	return " — " + note
}
