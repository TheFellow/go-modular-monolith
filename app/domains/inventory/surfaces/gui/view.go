package gui

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
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
	ControlBack         = "inventory-detail-back"
	ControlBreadcrumb   = "inventory-detail-breadcrumb"
)

type View struct {
	presenter                             *Presenter
	root, browse, detail, mutation        *framework.Container
	list                                  *widget.Table
	listStack, empty                      *framework.Container
	expression, amount, cost              *ui.SemanticEntry
	tags                                  *ui.TagTokenEditor
	limit, reason                         *widget.Select
	save, cancel, refresh, previous, next *ui.SemanticButton
	adjust, set, tagAction                *ui.SemanticButton
	status, formStatus, title, crumb      *widget.Label
	state                                 State
	rows                                  map[string]*ui.SemanticButton
	rendering                             bool
	renderedMode                          Mode
	renderedForm                          Form
	renderedInstance                      uint64
}

var _ ui.View = (*View)(nil)
var _ ui.Activated = (*View)(nil)

func NewView(p *Presenter) *View {
	v := &View{presenter: p, state: p.Snapshot(), rows: map[string]*ui.SemanticButton{}}
	v.limit = widget.NewSelect([]string{"25", "50", "100"}, nil)
	v.limit.SetSelected(strconv.Itoa(v.state.Limit))
	bar := ui.NewSingleRowFilterBar(ControlFilter, ControlApplyFilter, `Filter inventory (for example: tags contains "featured")`, v.state.Expression,
		[]ui.FilterPreset{{ID: "inventory-stock", Placeholder: "Stock", Options: []ui.FilterOption{{Label: "All stock"}, {Label: "Low stock", Expression: fmt.Sprintf("quantity <= %g", v.state.LowStock)}}}},
		container.NewBorder(nil, nil, widget.NewLabel("Page size"), nil, v.limit), func(expression string) {
			limit, _ := strconv.Atoi(v.limit.Selected)
			p.Filter(AllStock, expression, v.state.LowStock, limit)
		})
	v.expression = bar.Expression
	// Mirrors the CLI inventory list except for the inventory record ID; the
	// joined ingredient name makes the otherwise opaque ingredient ID useful.
	columns := []string{"Ingredient", "Ingredient ID", "Quantity", "Unit", "Cost per unit", "Last updated", "Tags", "Status", "Actions"}
	v.list = ui.NewRowTable(func() (int, int) { return len(v.state.Rows) + 1, len(columns) }, func() framework.CanvasObject {
		return ui.NewActionCell()
	}, func(id widget.TableCellID, object framework.CanvasObject) {
		cell := object
		if id.Row == 0 {
			ui.ShowCellText(cell, columns[id.Col], true)
			return
		}
		r := v.state.Rows[id.Row-1]
		values := []string{r.Ingredient.Name, r.Inventory.IngredientID.String(), fmt.Sprintf("%.2f", r.Inventory.Amount.Value()), string(r.Inventory.Amount.Unit()), r.Cost, formatInventoryTime(r.Inventory.LastUpdated), r.Inventory.Tags.Canonical().String(), r.Status}
		if id.Col == len(columns)-1 {
			rowID := r.Inventory.ID
			actions := []ui.RowAction{{Label: "View", Run: func() { p.Select(rowID) }}}
			if r.CanAdjust {
				actions = append(actions, ui.RowAction{Label: "Adjust", Run: func() { p.Select(rowID); p.StartAdjust() }})
			}
			if r.CanSet {
				actions = append(actions, ui.RowAction{Label: "Set", Run: func() { p.Select(rowID); p.StartSet() }})
			}
			if r.CanTag {
				actions = append(actions, ui.RowAction{Label: "Tags", Run: func() { p.Select(rowID); p.StartTags() }})
			}
			ui.ShowCellActions(cell, actions)
			return
		}
		if id.Col == 6 {
			ui.ShowCellTags(cell, values[id.Col])
			return
		}
		ui.ShowCellText(cell, values[id.Col], false)
	})
	v.list.OnSelected = func(id widget.TableCellID) {
		if id.Row > 0 && id.Col < len(columns)-1 {
			v.list.UnselectAll()
			p.Select(v.state.Rows[id.Row-1].Inventory.ID)
		}
	}
	for i, width := range []float32{180, 230, 90, 70, 120, 210, 180, 70} {
		v.list.SetColumnWidth(i, width)
	}
	v.list.SetColumnWidth(8, 120)
	v.empty = ui.EmptyCollection(ui.IconEmpty, "No inventory found", "Adjust the filter to find stock items.")
	v.listStack = container.NewStack(v.list, v.empty)
	v.refresh = ui.WithIcon(ui.NewButton(ControlRefresh, "Refresh", p.Load), ui.IconRefresh)
	v.previous = ui.WithIcon(ui.NewButton(ControlPrevious, "Previous", p.PreviousPage), ui.IconPrevious)
	v.next = ui.WithIcon(ui.NewButton(ControlNext, "Next", p.NextPage), ui.IconNext)
	v.status = widget.NewLabel("")
	v.browse = ui.StandardListPage(ui.ListPage{Title: "Inventory", Subtitle: "Review stock levels and select an item for complete details.", Filters: bar.Content, CollectionActions: []framework.CanvasObject{v.refresh}, List: v.listStack, Status: v.status, Paging: container.NewHBox(v.previous, v.next), ListRatio: .35}).(*framework.Container)

	v.adjust = ui.Primary(ui.WithIcon(ui.NewButton(ControlAdjust, "Adjust stock", p.StartAdjust), ui.IconAdd))
	v.set = ui.WithIcon(ui.NewButton(ControlSet, "Set stock", p.StartSet), ui.IconSave)
	v.tagAction = ui.WithIcon(ui.NewButton(ControlTags, "Tags", p.StartTags), ui.IconTag)
	v.title, v.crumb, v.formStatus = widget.NewLabel("Inventory item"), widget.NewLabel(""), widget.NewLabel("")
	v.detail = ui.StandardFormPage(ui.FormPage{Title: "Inventory item", Breadcrumb: v.breadcrumb(""), Fields: container.NewStack(), Status: v.formStatus}).(*framework.Container)

	v.amount, v.cost = ui.NewEntry(ControlAmount), ui.NewEntry(ControlCost)
	v.tags = ui.NewTagTokenEditor(ControlFormTags, "")
	v.tags.Normalize = tag.UpsertCollection
	v.reason = widget.NewSelect([]string{string(inventorymodels.ReasonReceived), string(inventorymodels.ReasonUsed), string(inventorymodels.ReasonSpilled), string(inventorymodels.ReasonExpired), string(inventorymodels.ReasonCorrected)}, nil)
	v.save = ui.WithIcon(ui.NewButton(ControlSave, "Save", func() { v.readForm(); p.Submit(p.Snapshot().Form) }), ui.IconSave)
	v.cancel = ui.WithIcon(ui.NewButton(ControlCancel, "Cancel", p.Cancel), ui.IconCancel)
	v.mutation = ui.StandardFormPage(ui.FormPage{Title: "Inventory item", Breadcrumb: v.breadcrumb(""), Fields: v.mutationFields(Adjust), Status: v.formStatus, Save: v.save, Cancel: v.cancel}).(*framework.Container)
	v.root = container.NewStack(v.browse)
	v.amount.OnChanged = func(string) { v.changed() }
	v.cost.OnChanged = func(string) { v.changed() }
	v.tags.OnChanged = func(string) { v.changed() }
	v.reason.OnChanged = func(string) { v.changed() }
	p.OnChange(v.render)
	v.render(v.state)
	return v
}

func (v *View) breadcrumb(name string) framework.CanvasObject {
	return container.NewHBox(ui.WithIcon(ui.NewButton(ControlBack, "Back", v.presenter.Back), ui.IconBack), ui.NewButton(ControlBreadcrumb, "Inventory", v.presenter.ResetList), widget.NewLabel(">"), widget.NewLabel(name))
}

func (v *View) Title() string                   { return "Inventory" }
func (v *View) Content() framework.CanvasObject { return v.root }
func (v *View) Activate()                       { v.presenter.ResetList() }
func (v *View) HasUnsavedChanges() bool         { return v.presenter.Snapshot().Dirty }
func (v *View) ExecuteCommand(c ui.Command) bool {
	s := v.presenter.Snapshot()
	switch c {
	case ui.CommandRefresh:
		return s.Mode == Browse && ui.Trigger(v.refresh)
	case ui.CommandSave:
		return (s.Mode == Adjust || s.Mode == Set || s.Mode == Tags) && ui.Trigger(v.save)
	case ui.CommandCancel:
		return s.Mode != Browse && ui.Trigger(v.cancel)
	case ui.CommandNew:
		return false
	}
	return false
}

func (v *View) mutationFields(mode Mode) framework.CanvasObject {
	switch mode {
	case Adjust:
		return ui.DetailForm(ui.DetailField("Signed amount", v.amount), ui.DetailField("Cost per unit", v.cost), ui.DetailField("Reason", v.reason), ui.DetailField("Tags", v.tags.Content))
	case Set:
		return ui.DetailForm(ui.DetailField("Quantity", v.amount), ui.DetailField("Cost per unit", v.cost), ui.DetailField("Tags", v.tags.Content))
	case Tags:
		return ui.DetailForm(ui.DetailField("Tags", v.tags.Content))
	default:
		return container.NewVBox()
	}
}

func (v *View) detailFields(s State) framework.CanvasObject {
	if s.Selected == nil {
		return container.NewVBox()
	}
	r := s.Selected
	entry := func(value string) *widget.Entry {
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
	form := ui.DetailForm(ui.DetailField("Ingredient", entry(r.Ingredient.Name)), ui.DetailField("Category", entry(string(r.Ingredient.Category))), ui.DetailField("Quantity", entry(r.Quantity)), ui.DetailField("Cost per unit", entry(r.Cost)), ui.DetailField("Status", entry(r.Status)), ui.DetailField("Tags", ui.TagPillsCSV(r.Inventory.Tags.Canonical().String())), ui.DetailField("Last updated", entry(formatInventoryTime(r.Inventory.LastUpdated))))
	return container.NewVBox(ui.ActionBar(nil, []framework.CanvasObject{v.adjust, v.set, v.tagAction}), form)
}

func (v *View) changed() {
	if v.rendering {
		return
	}
	s := v.presenter.Snapshot()
	if s.Mode != Adjust && s.Mode != Set && s.Mode != Tags {
		return
	}
	v.readForm()
}
func (v *View) readForm() {
	s := v.presenter.Snapshot()
	f := Form{Amount: v.amount.Text, Cost: v.cost.Text, Tags: v.tags.CSV(), Reason: inventorymodels.AdjustmentReason(v.reason.Selected), ReplaceTags: s.Mode != Tags}
	v.presenter.SetForm(f)
}
func (v *View) populate(f Form) {
	v.rendering = true
	defer func() { v.rendering = false }()
	v.amount.SetText(f.Amount)
	v.cost.SetText(f.Cost)
	v.tags.SetCSV(f.Tags)
	v.reason.SetSelected(string(f.Reason))
}

func (v *View) render(s State) {
	v.rendering = true
	defer func() { v.rendering = false }()
	v.state = s
	v.rows = make(map[string]*ui.SemanticButton, len(s.Rows))
	for i := range s.Rows {
		r := s.Rows[i]
		id := r.Inventory.ID
		button := ui.NewButton(ControlSelectPrefix+id.String(), r.Ingredient.Name, func() { v.presenter.Select(id) })
		if s.Mode != Browse {
			button.Disable()
		}
		v.rows[id.String()] = button
	}
	if s.Mode == Adjust || s.Mode == Set || s.Mode == Tags {
		if v.renderedMode != s.Mode || v.renderedInstance != s.FormInstance || !reflect.DeepEqual(v.renderedForm, s.Form) {
			v.populate(s.Form)
			v.renderedMode = s.Mode
			v.renderedInstance = s.FormInstance
			v.renderedForm = s.Form
			name := "Inventory item"
			if s.Selected != nil {
				name = s.Selected.Ingredient.Name
			}
			v.mutation = ui.StandardFormPage(ui.FormPage{Title: name, Breadcrumb: v.breadcrumb(name), Fields: v.mutationFields(s.Mode), Status: v.formStatus, Save: v.save, Cancel: v.cancel}).(*framework.Container)
		}
	}
	if s.Selected != nil {
		v.title.SetText(s.Selected.Ingredient.Name)
		v.crumb.SetText(s.Selected.Ingredient.Name)
		if s.Mode == Viewing {
			v.detail = ui.StandardFormPage(ui.FormPage{Title: s.Selected.Ingredient.Name, Breadcrumb: v.breadcrumb(s.Selected.Ingredient.Name), Fields: v.detailFields(s), Status: v.formStatus}).(*framework.Container)
		}
	}
	v.adjust.Hidden = s.Selected == nil || !s.CanAdjust
	v.set.Hidden = s.Selected == nil || !s.CanSet
	v.tagAction.Hidden = s.Selected == nil || !s.CanTag
	if len(s.History) == 0 || s.Status == ui.Loading {
		v.previous.Disable()
	} else {
		v.previous.Enable()
	}
	if s.Next == "" || s.Status == ui.Loading {
		v.next.Disable()
	} else {
		v.next.Enable()
	}
	if s.Submitting || !s.Dirty {
		v.save.Disable()
		v.cancel.Disable()
	} else {
		v.save.Enable()
		v.cancel.Enable()
	}
	v.tags.SetEnabled(!s.Submitting)
	v.empty.Hidden = s.Status != ui.Loaded || len(s.Rows) != 0
	v.list.Hidden = s.Status == ui.Loaded && len(s.Rows) == 0
	if s.Submitting {
		v.formStatus.SetText("Saving…")
	} else if s.Err != nil {
		v.formStatus.SetText("Error: " + s.Err.Error())
	} else {
		v.formStatus.SetText("")
	}
	if s.Status == ui.Loading {
		v.status.SetText("Loading inventory…")
	} else if s.Err != nil {
		v.status.SetText("Error: " + s.Err.Error())
	} else {
		v.status.SetText(fmt.Sprintf("%d inventory items", len(s.Rows)))
	}
	v.list.Refresh()
	switch s.Mode {
	case Browse:
		v.root.Objects = []framework.CanvasObject{v.browse}
	case Viewing:
		v.root.Objects = []framework.CanvasObject{v.detail}
	case Adjust, Set, Tags:
		v.root.Objects = []framework.CanvasObject{v.mutation}
	}
	v.root.Refresh()
}

func formatInventoryTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}

func inventoryDetailLabels(r *Row, tags string) []string {
	return []string{r.Ingredient.Name, "Ingredient ID: " + r.Ingredient.ID.String(), "Inventory ID: " + r.Inventory.ID.String(), "Category: " + string(r.Ingredient.Category), "Unit: " + string(r.Ingredient.Unit), "Tags: " + tags, "Quantity: " + r.Quantity, "Cost per unit: " + r.Cost, "Status: " + r.Status, "Last updated: " + formatInventoryTime(r.Inventory.LastUpdated)}
}
