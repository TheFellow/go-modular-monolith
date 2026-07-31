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

type semanticSelect struct {
	widget.Select
	id string
}

func newSemanticSelect(id string, options []string) *semanticSelect {
	s := &semanticSelect{id: id}
	s.Options = options
	s.ExtendBaseWidget(s)
	return s
}
func (s *semanticSelect) SemanticID() string { return s.id }

type View struct {
	presenter                                                                *Presenter
	root                                                                     *framework.Container
	expression, menuQuery, drinkQuery, quantity, itemNotes, orderNotes, tags *ui.SemanticEntry
	limit                                                                    *semanticSelect
	menus, drinks                                                            *widget.Select
	list                                                                     *widget.Table
	rows                                                                     map[string]*ui.SemanticButton
	removeItems                                                              map[int]*ui.SemanticButton
	refresh, create, save, cancel                                            *ui.SemanticButton
	state                                                                    State
	rendering                                                                bool
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
	default:
		body = v.form(s)
	}
	v.root.Objects = []framework.CanvasObject{body}
	v.root.Refresh()
}

func (v *View) browser(s State) framework.CanvasObject {
	v.limit = newSemanticSelect("orders-filter-limit", []string{"25", "50", "100"})
	v.limit.SetSelected(strconv.Itoa(s.Filter.Limit))
	bar := ui.NewSingleRowFilterBar(ControlFilter, ControlApplyFilter, `Filter orders (for example: tags contains "featured")`, s.Filter.Expression,
		[]ui.FilterPreset{{ID: "orders-status", Placeholder: "Status", Options: []ui.FilterOption{{Label: "Any status"}, {Label: "Pending", Expression: `status == "pending"`}, {Label: "Completed", Expression: `status == "completed"`}, {Label: "Cancelled", Expression: `status == "cancelled"`}}}},
		container.NewBorder(nil, nil, widget.NewLabel("Page size"), nil, v.limit), func(expression string) {
			limit, _ := strconv.Atoi(v.limit.Selected)
			v.presenter.ApplyFilter(Filter{Expression: expression, Limit: limit})
		})
	v.expression = bar.Expression
	v.refresh = ui.WithIcon(ui.NewButton(ControlRefresh, "Refresh", v.presenter.Refresh), ui.IconRefresh)
	v.create = ui.Primary(ui.WithIcon(ui.NewButton(ControlPlace, "Place order", v.presenter.StartPlace), ui.IconAdd))
	previous := ui.WithIcon(ui.NewButton(ControlPrevious, "Previous", v.presenter.PreviousPage), ui.IconPrevious)
	next := ui.WithIcon(ui.NewButton(ControlNext, "Next", v.presenter.NextPage), ui.IconNext)
	busy := s.Loading || s.Submitting || s.Confirming
	setEnabled(previous, !busy && len(s.History) > 0)
	setEnabled(next, !busy && s.Next != "")
	setEnabled(v.refresh, !busy)
	setEnabled(v.create, !busy)
	v.create.Hidden = !s.CanPlace
	setEnabled(v.expression, !busy)
	setEnabled(v.limit, !busy)
	setEnabled(bar.Apply, !busy)
	columns := []string{"Menu", "Menu ID", "Status", "Items", "Total quantity", "Total", "Created", "Completed", "Tags"}
	v.list = widget.NewTable(func() (int, int) { return len(v.state.Rows) + 1, len(columns) }, func() framework.CanvasObject {
		l := widget.NewLabel("")
		l.Truncation = framework.TextTruncateEllipsis
		return l
	}, func(id widget.TableCellID, object framework.CanvasObject) {
		l := object.(*widget.Label)
		if id.Row == 0 {
			l.TextStyle = framework.TextStyle{Bold: true}
			l.SetText(columns[id.Col])
			return
		}
		l.TextStyle = framework.TextStyle{}
		row := v.state.Rows[id.Row-1]
		completed := ""
		if t, ok := row.Order.CompletedAt.Unwrap(); ok {
			completed = formatTime(t)
		}
		qty := 0
		for _, item := range row.Order.Items {
			qty += item.Quantity
		}
		values := []string{row.MenuName, row.Order.MenuID.String(), string(row.Order.Status), strconv.Itoa(len(row.Order.Items)), strconv.Itoa(qty), row.Total, formatTime(row.Order.CreatedAt), completed, row.Order.Tags.Canonical().String()}
		l.SetText(values[id.Col])
	})
	v.list.OnSelected = func(id widget.TableCellID) {
		if id.Row > 0 {
			v.list.UnselectAll()
			v.presenter.Select(id.Row - 1)
		}
	}
	for i, width := range []float32{190, 240, 100, 65, 100, 90, 175, 175, 180} {
		v.list.SetColumnWidth(i, width)
	}
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
	return ui.StandardListPage(ui.ListPage{Title: "Orders", Subtitle: "Browse orders and select one for complete details.", Filters: bar.Content, CollectionActions: []framework.CanvasObject{v.create, v.refresh}, List: list, Status: widget.NewLabel(status), Paging: container.NewHBox(previous, next)})
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
	fields := container.NewVBox(widget.NewForm(
		widget.NewFormItem("Order ID", readonly(r.Order.ID.String())), widget.NewFormItem("Menu", readonly(r.MenuName)),
		widget.NewFormItem("Status", readonly(string(r.Order.Status))), widget.NewFormItem("Created", readonly(formatTime(r.Order.CreatedAt))),
		widget.NewFormItem("Completed", readonly(completed)), widget.NewFormItem("Tags", readonly(r.Order.Tags.Canonical().String())),
		widget.NewFormItem("Notes", notes)), widget.NewLabelWithStyle("Items", framework.TextAlignLeading, framework.TextStyle{Bold: true}))
	if len(r.Lines) == 0 {
		fields.Add(ui.EmptyCollection(ui.IconEmpty, "No order items", "This order contains no line items."))
	}
	for _, line := range r.Lines {
		fields.Add(widget.NewForm(widget.NewFormItem("Drink", readonly(line.Name)), widget.NewFormItem("Quantity", readonly(strconv.Itoa(line.Quantity))), widget.NewFormItem("Line total", readonly(line.Total)), widget.NewFormItem("Notes", readonly(line.Notes))))
	}
	fields.Add(widget.NewForm(widget.NewFormItem("Order total", readonly(r.Total))))
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
		items.Add(container.NewBorder(nil, nil, nil, remove, widget.NewLabel(fmt.Sprintf("%s × %d%s", item.Name, item.Quantity, noteSuffix(item.Notes)))))
	}
	if len(s.Form.Items) == 0 {
		items.Add(ui.EmptyCollection(ui.IconEmpty, "No items yet", "Choose an available drink and add it to the order."))
	}
	v.orderNotes = ui.NewEntry(ControlOrderNotes)
	v.orderNotes.MultiLine = true
	v.orderNotes.SetPlaceHolder("Order notes (optional)")
	v.orderNotes.SetText(s.Form.Notes)
	v.orderNotes.OnChanged = v.presenter.SetPlaceNotes
	v.tags = ui.NewEntry(ControlPlaceTags)
	v.tags.SetText(s.Form.Tags)
	v.tags.OnChanged = v.presenter.SetPlaceTags
	v.save = ui.WithIcon(ui.NewButton(ControlPlaceSave, "Place", func() {
		v.presenter.SetPlaceNotes(v.orderNotes.Text)
		v.presenter.SetPlaceTags(v.tags.Text)
		v.presenter.SavePlace()
	}), ui.IconSave)
	v.cancel = ui.WithIcon(ui.NewButton(ControlFormCancel, "Cancel", v.presenter.CancelForm), ui.IconCancel)
	busy := s.Submitting || s.CatalogLoading
	if busy {
		for _, b := range []*ui.SemanticButton{searchMenus, searchDrinks, add, v.save, v.cancel} {
			b.Disable()
		}
		for _, e := range []*ui.SemanticEntry{v.menuQuery, v.drinkQuery, v.quantity, v.itemNotes, v.orderNotes, v.tags} {
			e.Disable()
		}
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
	fields := container.NewVBox(container.NewBorder(nil, nil, nil, searchMenus, v.menuQuery), widget.NewLabel("Published menu"), v.menus, container.NewBorder(nil, nil, nil, searchDrinks, v.drinkQuery), widget.NewLabel("Available drink"), v.drinks, widget.NewLabel("Quantity"), v.quantity, widget.NewLabel("Item notes"), v.itemNotes, add, widget.NewLabelWithStyle("Order items", framework.TextAlignLeading, framework.TextStyle{Bold: true}), items, widget.NewLabel("Order notes"), v.orderNotes, widget.NewLabel("Tags (complete set)"), v.tags)
	return ui.StandardFormPage(ui.FormPage{Title: "Place order", Breadcrumb: container.NewHBox(ui.WithIcon(ui.NewButton(ControlBack, "Back", v.presenter.Back), ui.IconBack), ui.NewButton(ControlBreadcrumb, "Orders", v.presenter.ResetList), widget.NewLabel(">"), widget.NewLabel("Place order")), Subtitle: "Choose a published menu, add drinks, then place the order.", Fields: fields, Status: widget.NewLabel(message), Save: v.save, Cancel: v.cancel})
}
func (v *View) tagForm(s State) framework.CanvasObject {
	v.tags = ui.NewEntry(ControlTagValues)
	v.tags.SetPlaceHolder("featured, region=west")
	v.rendering = true
	v.tags.SetText(s.Form.Tags)
	v.rendering = false
	v.tags.OnChanged = func(value string) {
		if !v.rendering {
			v.presenter.SetTagForm(value)
		}
	}
	v.save = ui.WithIcon(ui.NewButton(ControlTagSave, "Save", func() { v.presenter.SaveTags(v.tags.Text) }), ui.IconSave)
	v.cancel = ui.WithIcon(ui.NewButton(ControlFormCancel, "Cancel", v.presenter.CancelForm), ui.IconCancel)
	setEnabled(v.save, !s.Submitting && s.Dirty)
	setEnabled(v.cancel, !s.Submitting && s.Dirty)
	setEnabled(v.tags, !s.Submitting)
	message := ""
	if s.Err != nil {
		message = "Error: " + s.Err.Error()
	}
	title := orderTitle(s.Selected)
	return ui.StandardFormPage(ui.FormPage{Title: "Edit order tags", Breadcrumb: v.breadcrumb(title), Subtitle: "Complete tag set (CSV); clear to remove all tags.", Fields: v.tags, Status: widget.NewLabel(message), Save: v.save, Cancel: v.cancel})
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
