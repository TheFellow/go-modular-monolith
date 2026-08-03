package gui

import (
	"fmt"
	"strconv"
	"strings"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
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
	ControlSelectPrefix     = "orders-select-"
	ControlBack             = "orders-detail-back"
	ControlBreadcrumb       = "orders-detail-breadcrumb"
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
	presenter                                                          *Presenter
	root                                                               *framework.Container
	expression, menuQuery, drinkQuery, quantity, itemNotes, orderNotes *ui.SemanticEntry
	tags                                                               *ui.TagTokenEditor
	menus, drinks                                                      *widget.Select
	list                                                               *widget.Table
	removeItems                                                        map[int]*ui.SemanticButton
	refresh, create, save, cancel                                      *ui.SemanticButton
	state                                                              State
	rendering                                                          bool
	tagNaturalWidth                                                    float32
}

var _ ui.View = (*View)(nil)
var _ ui.Activated = (*View)(nil)

func NewView(p *Presenter) *View {
	v := &View{presenter: p, root: container.NewStack(), state: p.State()}
	p.Observe(v.render)
	return v
}
func (v *View) Title() string                   { return "Orders" }
func (v *View) Content() framework.CanvasObject { return v.root }
func (v *View) Activate()                       { v.presenter.ResetList() }
func (v *View) HasUnsavedChanges() bool         { return v.presenter.State().Dirty }
func (v *View) ExecuteCommand(c ui.Command) bool {
	s := v.presenter.State()
	switch c {
	case ui.CommandRefresh:
		return s.Mode == Browsing && ui.Trigger(v.refresh)
	case ui.CommandNew:
		return s.Mode == Browsing && ui.Trigger(v.create)
	case ui.CommandSave:
		return (s.Mode == Placing || s.Mode == Tagging) && ui.Trigger(v.save)
	case ui.CommandCancel:
		return s.Mode != Browsing && ui.Trigger(v.cancel)
	}
	return false
}
func (v *View) render(s State) {
	v.state = s
	var body framework.CanvasObject
	switch s.Mode {
	case Browsing:
		body = v.browser(s)
	case Viewing:
		body = v.detail(s)
	case Placing, Tagging:
		body = v.form(s)
	}
	v.root.Objects = []framework.CanvasObject{body}
	v.root.Refresh()
}

func (v *View) browser(s State) framework.CanvasObject {
	bar := ui.NewSingleRowFilterBar(ControlFilter, ControlApplyFilter, `Filter orders (for example: tags contains "featured")`, s.Filter.Expression,
		[]ui.FilterPreset{{ID: "orders-status", Placeholder: "Status", Options: []ui.FilterOption{{Label: "Any status"}, {Label: "Pending", Expression: `status == "pending"`}, {Label: "Completed", Expression: `status == "completed"`}, {Label: "Cancelled", Expression: `status == "cancelled"`}}}},
		nil, func(expression string) { v.presenter.ApplyFilter(Filter{Expression: expression, Limit: ui.PageLimit}) })
	v.expression = bar.Expression
	v.refresh = ui.WithIcon(ui.NewButton(ControlRefresh, "Refresh", v.presenter.Refresh), ui.IconRefresh)
	v.create = ui.Primary(ui.WithIcon(ui.NewButton(ControlPlace, "Place order", v.presenter.StartPlace), ui.IconAdd))
	busy := s.Loading || s.Submitting || s.Confirming
	setEnabled(v.refresh, !busy)
	setEnabled(v.create, !busy)
	v.create.Hidden = !s.CanPlace
	setEnabled(v.expression, !busy)
	setEnabled(bar.Apply, !busy)
	columns := []string{"Menu", "Menu ID", "Status", "Items", "Total quantity", "Total", "Created", "Completed", "Tags", "Actions"}
	if v.list == nil {
		v.list = ui.NewAutoPagingRowTable(func() (int, int) { return len(v.state.Rows), len(columns) }, func() framework.CanvasObject {
			return ui.NewActionCell()
		}, func(id widget.TableCellID, object framework.CanvasObject) {
			cell := object
			row := v.state.Rows[id.Row]
			completed := ""
			if t, ok := row.Order.CompletedAt.Unwrap(); ok {
				completed = formatTime(t)
			}
			qty := 0
			for _, item := range row.Order.Items {
				qty += item.Quantity
			}
			values := []string{row.MenuName, row.Order.MenuID.String(), string(row.Order.Status), strconv.Itoa(len(row.Order.Items)), strconv.Itoa(qty), row.Total, formatTime(row.Order.CreatedAt), completed, row.Order.Tags.Canonical().String()}
			if id.Col == len(columns)-1 {
				index := id.Row
				actions := []ui.RowAction{{Label: "View", Run: func() { v.presenter.Select(index) }}}
				canComplete, canCancel, canTag := v.presenter.ListPermissions(index)
				if canComplete {
					actions = append(actions, ui.RowAction{Label: "Complete", Run: func() { v.presenter.Select(index); v.presenter.ConfirmComplete() }})
				}
				if canCancel {
					actions = append(actions, ui.RowAction{Label: "Cancel order", Run: func() { v.presenter.Select(index); v.presenter.ConfirmCancel() }})
				}
				if canTag {
					actions = append(actions, ui.RowAction{Label: "Tags", Run: func() { v.presenter.Select(index); v.presenter.StartTags() }})
				}
				ui.ShowCellActions(cell, actions)
				return
			}
			if id.Col == 8 {
				ui.ShowCellTags(cell, values[id.Col])
				return
			}
			ui.ShowCellText(cell, values[id.Col], false)
		}, func() { framework.Do(v.presenter.NextPage) })
		v.list.OnSelected = func(id widget.TableCellID) {
			if id.Row >= 0 && id.Col < len(columns)-1 {
				v.list.UnselectAll()
				v.presenter.Select(id.Row)
			}
		}
		ui.ConfigureRowTable(v.list, []ui.TableColumn{{Title: "Menu", Width: 190}, {Title: "Menu ID", Width: 240}, {Title: "Status", Width: 100}, {Title: "Items", Width: 65}, {Title: "Total quantity", Width: 100}, {Title: "Total", Width: 90}, {Title: "Created", Width: 175}, {Title: "Completed", Width: 175}, {Title: "Tags", Width: 180}, {Title: "Actions", Width: 140}}, nil)
	}
	if len(s.Rows) > 0 {
		values := make([]string, len(s.Rows))
		for i, row := range s.Rows {
			values[i] = row.Order.Tags.Canonical().String()
		}
		if width := ui.TagPillColumnWidth(values, 180); width > v.tagNaturalWidth {
			v.list.SetColumnWidth(8, width)
			v.tagNaturalWidth = width
		}
	}
	v.list.Refresh()
	list := framework.CanvasObject(v.list)
	if len(s.Rows) == 0 && !s.Loading {
		list = ui.EmptyCollection(ui.IconEmpty, "No orders found", "Adjust the filter or place an order.")
	}
	status := ""
	if s.Loading {
		status = "Loading orders…"
	} else if s.Err != nil {
		status = "Error: " + s.Err.Error()
	} else {
		status = fmt.Sprintf("%d orders", len(s.Rows))
	}
	return ui.StandardListPage(ui.ListPage{Title: "Orders", Subtitle: "Browse orders and select one for complete details.", Filters: bar.Content, CollectionActions: []framework.CanvasObject{v.create, v.refresh}, List: list, Status: widget.NewLabel(status)})
}

func (v *View) breadcrumb(name string) framework.CanvasObject {
	return container.NewHBox(ui.WithIcon(ui.NewButton(ControlBack, "Back", v.presenter.Back), ui.IconBack), ui.NewButton(ControlBreadcrumb, "Orders", v.presenter.ResetList), widget.NewLabel(">"), widget.NewLabel(name))
}
func orderTitle(r *Row) string {
	if r == nil {
		return "Order"
	}
	return "Order " + r.Order.ID.String()
}
func readonly(value string) *widget.Entry {
	e := widget.NewEntry()
	restoring := false
	e.OnChanged = func(changed string) {
		if restoring || changed == value {
			return
		}
		restoring = true
		e.SetText(value)
		restoring = false
	}
	e.SetText(value)
	return e
}
func (v *View) detail(s State) framework.CanvasObject {
	if s.Selected == nil {
		return ui.StandardFormPage(ui.FormPage{Title: "Order", Breadcrumb: v.breadcrumb("Order"), Fields: ui.EmptyDetail("an order")})
	}
	r := s.Selected
	title := orderTitle(r)
	completed := ""
	if t, ok := r.Order.CompletedAt.Unwrap(); ok {
		completed = formatTime(t)
	}
	notes := readonly(r.Order.Notes)
	notes.MultiLine = true
	fields := container.NewVBox(ui.DetailForm(
		ui.DetailField("Order ID", readonly(r.Order.ID.String())), ui.DetailField("Menu", readonly(r.MenuName)),
		ui.DetailField("Status", readonly(string(r.Order.Status))), ui.DetailField("Created", readonly(formatTime(r.Order.CreatedAt))),
		ui.DetailField("Completed", readonly(completed)), ui.DetailField("Tags", ui.TagPillsCSV(r.Order.Tags.Canonical().String())),
		ui.DetailField("Notes", notes)), widget.NewLabelWithStyle("Items", framework.TextAlignLeading, framework.TextStyle{Bold: true}))
	if len(r.Lines) == 0 {
		fields.Add(ui.EmptyCollection(ui.IconEmpty, "No order items", "This order contains no line items."))
	}
	for _, line := range r.Lines {
		name := widget.NewLabelWithStyle(line.Name, framework.TextAlignLeading, framework.TextStyle{Bold: true})
		meta := fmt.Sprintf("Quantity %d  ·  %s", line.Quantity, line.Total)
		if strings.TrimSpace(line.Notes) != "" {
			meta += "  ·  " + line.Notes
		}
		fields.Add(container.NewVBox(container.NewBorder(nil, nil, nil, widget.NewLabel(meta), name), widget.NewSeparator()))
	}
	fields.Add(ui.DetailForm(ui.DetailField("Order total", readonly(r.Total))))
	actions := []framework.CanvasObject{}
	cleanPending := !s.Submitting && !s.Confirming && !s.Dirty && r.Order.Status == models.OrderStatusPending
	if cleanPending && s.CanComplete {
		actions = append(actions, ui.NewButton(ControlComplete, "Complete", v.presenter.ConfirmComplete))
	}
	if !s.Submitting && !s.Confirming && !s.Dirty && s.CanTag {
		actions = append(actions, ui.WithIcon(ui.NewButton(ControlTags, "Tags", v.presenter.StartTags), ui.IconTag))
	}
	if cleanPending && s.CanCancel {
		actions = append(actions, ui.Destructive(ui.NewButton(ControlCancelOrder, "Cancel order", v.presenter.ConfirmCancel)))
	}
	if actionBar := ui.ActionBar(nil, actions); actionBar != nil {
		fields.Objects = append([]framework.CanvasObject{actionBar}, fields.Objects...)
	}
	status := ""
	if s.Err != nil {
		status = "Error: " + s.Err.Error()
	}
	return ui.StandardFormPage(ui.FormPage{Title: title, Breadcrumb: v.breadcrumb(title), Fields: fields, Status: widget.NewLabel(status)})
}

func (v *View) form(s State) framework.CanvasObject {
	if s.Mode == Tagging {
		return v.tagForm(s)
	}
	return v.placeForm(s)
}
func (v *View) placeForm(s State) framework.CanvasObject {
	v.menuQuery = ui.NewEntry(ControlMenuSearch)
	v.menuQuery.SetPlaceHolder("Search published menus")
	v.menuQuery.SetText(s.Form.MenuQuery)
	searchMenus := ui.NewButton(ControlMenuSearchApply, "Search", func() { v.presenter.SearchMenus(v.menuQuery.Text) })
	ui.SubmitOnEnter(v.menuQuery, searchMenus)
	menuLabels := make([]string, len(s.Menus))
	menuIDs := make(map[string]entity.MenuID, len(s.Menus))
	for i, m := range s.Menus {
		label := m.Name + "  [" + m.ID.String() + "]"
		menuLabels[i] = label
		menuIDs[label] = m.ID
	}
	v.menus = widget.NewSelect(menuLabels, nil)
	for _, m := range s.Menus {
		if m.ID == s.Form.MenuID {
			v.menus.SetSelected(m.Name + "  [" + m.ID.String() + "]")
		}
	}
	v.menus.OnChanged = func(value string) {
		if id, ok := menuIDs[value]; ok {
			v.presenter.ChooseMenu(id)
		}
	}
	v.drinkQuery = ui.NewEntry(ControlDrinkSearch)
	v.drinkQuery.SetPlaceHolder("Search available drinks")
	v.drinkQuery.SetText(s.Form.DrinkQuery)
	searchDrinks := ui.NewButton(ControlDrinkSearchApply, "Search", func() { v.presenter.SearchDrinks(v.drinkQuery.Text) })
	ui.SubmitOnEnter(v.drinkQuery, searchDrinks)
	drinkLabels := make([]string, len(s.Drinks))
	drinkIDs := make(map[string]entity.DrinkID, len(s.Drinks))
	for i, d := range s.Drinks {
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
	add := ui.WithIcon(ui.NewButton(ControlAddItem, "Add item", func() {
		qty, err := strconv.Atoi(strings.TrimSpace(v.quantity.Text))
		if err != nil {
			qty = 0
		}
		v.presenter.AddItem(drinkIDs[v.drinks.Selected], qty, v.itemNotes.Text)
	}), ui.IconAdd)
	items := container.NewVBox()
	v.removeItems = make(map[int]*ui.SemanticButton, len(s.Form.Items))
	for i, item := range s.Form.Items {
		index := i
		remove := ui.WithIcon(ui.NewButton(fmt.Sprintf("orders-place-remove-%d", i), "Remove", func() { v.presenter.RemoveItem(index) }), ui.IconDelete)
		if s.Submitting || s.CatalogLoading {
			remove.Disable()
		}
		v.removeItems[i] = remove
		remove.Hide()
		actions := ui.NewActionSelect([]string{"Remove"}, func(choice string) {
			if choice == "Remove" {
				v.presenter.RemoveItem(index)
			}
		})
		if s.Submitting || s.CatalogLoading {
			actions.Disable()
		}
		name := widget.NewLabelWithStyle(item.Name, framework.TextAlignLeading, framework.TextStyle{Bold: true})
		meta := widget.NewLabel(fmt.Sprintf("Quantity %d%s", item.Quantity, noteSuffix(item.Notes)))
		line := container.NewBorder(nil, nil, nil, meta, name)
		items.Add(container.NewVBox(container.NewBorder(nil, nil, remove, container.NewCenter(actions), line), widget.NewSeparator()))
	}
	if len(s.Form.Items) == 0 {
		items.Add(ui.EmptyCollection(ui.IconEmpty, "No items yet", "Choose an available drink and add it to the order."))
	}
	v.orderNotes = ui.NewEntry(ControlOrderNotes)
	v.orderNotes.MultiLine = true
	v.orderNotes.SetPlaceHolder("Order notes (optional)")
	v.orderNotes.SetText(s.Form.Notes)
	v.orderNotes.OnChanged = v.presenter.SetPlaceNotes
	v.tags = ui.NewTagTokenEditor(ControlPlaceTags, s.Form.Tags)
	v.tags.Normalize = tag.UpsertCollection
	v.tags.OnChanged = v.presenter.SetPlaceTags
	v.save = ui.WithIcon(ui.NewButton(ControlPlaceSave, "Place", func() {
		v.presenter.SetPlaceNotes(v.orderNotes.Text)
		v.presenter.SetPlaceTags(v.tags.CSV())
		v.presenter.SavePlace()
	}), ui.IconSave)
	v.cancel = ui.WithIcon(ui.NewButton(ControlFormCancel, "Cancel", v.presenter.CancelForm), ui.IconCancel)
	busy := s.Submitting || s.CatalogLoading
	if busy {
		for _, b := range []*ui.SemanticButton{searchMenus, searchDrinks, add, v.save, v.cancel} {
			b.Disable()
		}
		for _, e := range []*ui.SemanticEntry{v.menuQuery, v.drinkQuery, v.quantity, v.itemNotes, v.orderNotes} {
			e.Disable()
		}
		v.tags.SetEnabled(false)
		v.menus.Disable()
		v.drinks.Disable()
	}
	message := ""
	if s.CatalogLoading {
		message = "Loading published menus…"
	}
	if s.Err != nil {
		message = "Error: " + s.Err.Error()
	}
	fields := container.NewVBox(container.NewBorder(nil, nil, nil, searchMenus, v.menuQuery), widget.NewLabel("Published menu"), v.menus, container.NewBorder(nil, nil, nil, searchDrinks, v.drinkQuery), widget.NewLabel("Available drink"), v.drinks, widget.NewLabel("Quantity"), v.quantity, widget.NewLabel("Item notes"), v.itemNotes, container.NewHBox(layout.NewSpacer(), add), widget.NewLabelWithStyle("Order items", framework.TextAlignLeading, framework.TextStyle{Bold: true}), items, widget.NewLabel("Order notes"), v.orderNotes, widget.NewLabel("Tags"), v.tags.Content)
	return ui.StandardFormPage(ui.FormPage{Title: "Place order", Breadcrumb: container.NewHBox(ui.WithIcon(ui.NewButton(ControlBack, "Back", v.presenter.Back), ui.IconBack), ui.NewButton(ControlBreadcrumb, "Orders", v.presenter.ResetList), widget.NewLabel(">"), widget.NewLabel("Place order")), Subtitle: "Choose a published menu, add drinks, then place the order.", Fields: fields, Status: widget.NewLabel(message), Save: v.save, Cancel: v.cancel})
}
func (v *View) tagForm(s State) framework.CanvasObject {
	v.tags = ui.NewTagTokenEditor(ControlTagValues, s.Form.Tags)
	v.tags.Normalize = tag.UpsertCollection
	v.tags.OnChanged = func(value string) {
		if !v.rendering {
			v.presenter.SetTagForm(value)
		}
	}
	v.save = ui.WithIcon(ui.NewButton(ControlTagSave, "Save", func() { v.presenter.SaveTags(v.tags.CSV()) }), ui.IconSave)
	v.cancel = ui.WithIcon(ui.NewButton(ControlFormCancel, "Cancel", v.presenter.CancelForm), ui.IconCancel)
	setEnabled(v.save, !s.Submitting && s.Dirty)
	setEnabled(v.cancel, !s.Submitting && s.Dirty)
	v.tags.SetEnabled(!s.Submitting)
	message := ""
	if s.Err != nil {
		message = "Error: " + s.Err.Error()
	}
	title := orderTitle(s.Selected)
	return ui.StandardFormPage(ui.FormPage{Title: "Edit order tags", Breadcrumb: v.breadcrumb(title), Subtitle: "Type a key or key=value and press Enter.", Fields: v.tags.Content, Status: widget.NewLabel(message), Save: v.save, Cancel: v.cancel})
}
func noteSuffix(note string) string {
	if strings.TrimSpace(note) == "" {
		return ""
	}
	return " — " + note
}
func setEnabled(o interface {
	Enable()
	Disable()
}, enabled bool) {
	if enabled {
		o.Enable()
	} else {
		o.Disable()
	}
}
