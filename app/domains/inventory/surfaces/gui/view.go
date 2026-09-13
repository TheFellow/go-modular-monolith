package gui

import (
	"cmp"
	"fmt"
	"reflect"
	"time"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	inventory "github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

const (
	ControlNew               = "inventory-new"
	ControlIngredient        = "inventory-new-ingredient"
	ControlFilter            = "inventory-filter"
	ControlThreshold         = "inventory-low-stock-threshold"
	ControlApplyFilter       = "inventory-apply-filter"
	ControlRefresh           = "inventory-refresh"
	ControlQuarantine        = "inventory-quarantine"
	ControlRelease           = "inventory-release"
	ControlDispose           = "inventory-dispose"
	ControlHistory           = "inventory-history"
	ControlCostUnit          = "inventory-form-cost-unit"
	ControlDispositionReason = "inventory-disposition-reason"
	ControlAdjust            = "inventory-adjust"
	ControlSet               = "inventory-set"
	ControlTags              = "inventory-tags"
	ControlSelectPrefix      = "inventory-select-"
	ControlAmount            = "inventory-form-amount"
	ControlQuantity          = "inventory-form-quantity"
	ControlCost              = "inventory-form-cost"
	ControlAdjustTags        = "inventory-adjust-tags"
	ControlSetTags           = "inventory-set-tags"
	ControlFormTags          = "inventory-form-tags"
	ControlSave              = "inventory-form-save"
	ControlCancel            = "inventory-form-cancel"
	ControlBack              = "inventory-detail-back"
	ControlBreadcrumb        = "inventory-detail-breadcrumb"
)

type View struct {
	create                                      *ui.SemanticButton
	ingredient                                  *widget.Select
	costUnit                                    *ui.SemanticEntry
	presenter                                   *Presenter
	root, browse, detail, mutation              *framework.Container
	list                                        *ui.RowTable
	listStack, empty                            *framework.Container
	expression, amount, cost                    *ui.SemanticEntry
	dispositionReason                           *ui.SemanticEntry
	tags                                        *ui.TagTokenEditor
	reason                                      *widget.Select
	save, cancel, refresh                       *ui.SemanticButton
	adjust, set, tagAction                      *ui.SemanticButton
	quarantine, release, dispose, historyAction *ui.SemanticButton
	status, formStatus, title, crumb            *widget.Label
	state                                       State
	rows                                        map[string]*ui.SemanticButton
	rendering                                   bool
	renderedMode                                Mode
	renderedForm                                Form
	renderedInstance                            uint64
}

var _ ui.View = (*View)(nil)
var _ ui.Activated = (*View)(nil)

func NewView(p *Presenter) *View {
	v := &View{presenter: p, state: p.Snapshot(), rows: map[string]*ui.SemanticButton{}}
	bar := ui.NewSingleRowFilterBar(ControlFilter, ControlApplyFilter, `Filter inventory (for example: tags contains "featured")`, v.state.Expression,
		[]ui.FilterPreset{{ID: "inventory-stock", Placeholder: "Stock", Options: []ui.FilterOption{{Label: "All stock"}, {Label: "Low stock", Expression: fmt.Sprintf("quantity <= %g", v.state.LowStock)}}}},
		nil, func(expression string) { p.Filter(AllStock, expression, v.state.LowStock, ui.PageLimit) })
	v.expression = bar.Expression
	columns := []string{"Ingredient", "On hand", "Available", "Cost", "Status", "Updated", "Tags", "Actions"}
	v.list = ui.NewAutoPagingRowTable(func() (int, int) { return len(v.state.Rows), len(columns) }, func() framework.CanvasObject {
		return ui.NewActionCell()
	}, func(id widget.TableCellID, object framework.CanvasObject) {
		cell := object
		r := v.state.Rows[id.Row]
		values := []string{r.Ingredient.Name, r.Inventory.Amount.String(), r.Inventory.Available().String(), r.Cost + " / " + string(r.Inventory.CostUnit), r.Status, ui.TableTimestamp(r.Inventory.LastUpdated), r.Inventory.Tags.Canonical().String()}
		if id.Col == len(columns)-1 {
			rowID := r.Inventory.ID
			actions := []ui.RowAction{{Label: "View", Run: func() { p.Select(rowID) }}}
			if actionEnabled(r.Actions, inventory.ControlAdjust) {
				actions = append(actions, ui.RowAction{Label: "Adjust", Run: func() { p.Select(rowID); p.StartAdjust() }})
			}
			if actionEnabled(r.Actions, inventory.ControlSet) {
				actions = append(actions, ui.RowAction{Label: "Set", Run: func() { p.Select(rowID); p.StartSet() }})
			}
			if actionVisible(r.Actions, inventory.ControlTags) {
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
	}, func() { framework.Do(p.NextPage) })
	v.list.OnSelected = func(id widget.TableCellID) {
		if id.Row >= 0 && id.Col < len(columns)-1 {
			v.list.UnselectAll()
			p.Select(v.state.Rows[id.Row].Inventory.ID)
		}
	}
	ui.ConfigureRowTable(v.list, []ui.TableColumn{{Title: "Ingredient", Width: 145, Flex: 2, Sortable: true}, {Title: "On hand", Width: 80, Flex: 1, Sortable: true}, {Title: "Available", Width: 80, Flex: 1, Sortable: true}, {Title: "Cost", Width: 75, Flex: 1, Sortable: true}, {Title: "Status", Width: 65, Flex: 1, Sortable: true}, {Title: "Updated", Width: 125, Flex: 1, Sortable: true}, {Title: "Tags", Width: 120, Flex: 2, Sortable: true}, {Title: "Actions", Width: ui.RowActionsWidth}}, func(column int, direction ui.SortDirection) {
		sortColumns := []int{0, 2, 4, 6, 9, 7, 8}
		p.SortRows(sortColumns[column], direction)
	})
	v.empty = ui.EmptyCollection(ui.IconEmpty, "No inventory found", "Adjust the filter to find stock items.")
	v.listStack = container.NewStack(v.list, v.empty)
	v.create = ui.WithIcon(ui.NewButton(ControlNew, "Receive new stock", p.StartNew), ui.IconAdd)
	v.refresh = ui.WithIcon(ui.NewButton(ControlRefresh, "Refresh", p.Load), ui.IconRefresh)
	v.status = widget.NewLabel("")
	v.browse = ui.StandardListPage(ui.ListPage{Title: "Inventory", Subtitle: "Review stock levels and select an item for complete details.", Filters: bar.Content, CollectionActions: []framework.CanvasObject{v.create, v.refresh}, List: v.listStack, Status: v.status, ListRatio: .35}).(*framework.Container)

	v.adjust = ui.Primary(ui.WithIcon(ui.NewButton(ControlAdjust, "Adjust stock", p.StartAdjust), ui.IconAdd))
	v.set = ui.WithIcon(ui.NewButton(ControlSet, "Set stock", p.StartSet), ui.IconSave)
	v.tagAction = ui.WithIcon(ui.NewButton(ControlTags, "Tags", p.StartTags), ui.IconTag)
	v.quarantine = ui.NewButton(ControlQuarantine, "Quarantine", p.StartQuarantine)
	v.release = ui.NewButton(ControlRelease, "Release quarantine", p.StartRelease)
	v.dispose = ui.NewButton(ControlDispose, "Dispose stock", p.StartDispose)
	v.historyAction = ui.NewButton(ControlHistory, "Stock history", p.ShowHistory)
	v.title, v.crumb, v.formStatus = widget.NewLabel("Inventory item"), widget.NewLabel(""), widget.NewLabel("")
	v.detail = ui.StandardFormPage(ui.FormPage{Title: "Inventory item", Breadcrumb: v.breadcrumb(""), Fields: container.NewStack(), Status: v.formStatus}).(*framework.Container)

	v.costUnit = ui.NewEntry(ControlCostUnit)
	v.costUnit.OnChanged = func(string) { v.changed() }
	v.dispositionReason = ui.NewEntry(ControlDispositionReason)
	v.dispositionReason.OnChanged = func(string) { v.changed() }
	v.amount, v.cost = ui.NewEntry(ControlAmount), ui.NewEntry(ControlCost)
	v.tags = ui.NewTagTokenEditor(ControlFormTags, "")
	v.tags.Normalize = tag.UpsertCollection
	v.reason = widget.NewSelect([]string{string(inventorymodels.ReasonReceived), string(inventorymodels.ReasonUsed), string(inventorymodels.ReasonSpilled), string(inventorymodels.ReasonExpired), string(inventorymodels.ReasonCorrected)}, nil)
	v.save = ui.WithIcon(ui.NewButton(ControlSave, "Save", func() { v.readForm(); p.Submit(p.Snapshot().Form) }), ui.IconSave)
	v.cancel = ui.WithIcon(ui.NewButton(ControlCancel, "Cancel", p.Cancel), ui.IconCancel)
	v.mutation = ui.StandardFormPage(ui.FormPage{Title: "Inventory item", Breadcrumb: v.breadcrumb(""), Fields: container.NewVBox(), Status: v.formStatus, Save: v.save, Cancel: v.cancel}).(*framework.Container)
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
	return container.NewHBox(ui.WithIcon(ui.NewButton(ControlBack, "Back", v.presenter.Back), ui.IconBack), ui.NewButton(ControlBreadcrumb, "Inventory", v.presenter.ResetList), widget.NewLabel("›"), widget.NewLabel(name))
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
		return isMutationMode(s.Mode) && ui.Trigger(v.save)
	case ui.CommandCancel:
		return s.Mode != Browse && ui.Trigger(v.cancel)
	case ui.CommandNew:
		return s.Mode == Browse && ui.Trigger(v.create)
	}
	return false
}

func (v *View) mutationFields(mode Mode) framework.CanvasObject {
	costLabel := "Cost per unit"
	if v.state.Selected != nil && v.state.Selected.Inventory.CostUnit != "" {
		costLabel = "Cost per " + string(v.state.Selected.Inventory.CostUnit)
	}
	switch mode {
	case Browse, Viewing, MovementHistory, SelectingIngredient:
		return container.NewVBox()
	case Adjust:
		return ui.DetailForm(ui.DetailField("Signed amount ("+string(v.state.Selected.Inventory.Amount.Unit())+")", v.amount), ui.DetailField(costLabel, v.cost), ui.DetailField("Cost unit", v.costUnit), ui.DetailField("Reason", v.reason), ui.DetailField("Tags", v.tags.Content))
	case Set:
		return ui.DetailForm(ui.DetailField("Quantity ("+string(v.state.Selected.Inventory.Amount.Unit())+")", v.amount), ui.DetailField(costLabel, v.cost), ui.DetailField("Cost unit", v.costUnit), ui.DetailField("Tags", v.tags.Content))
	case Quarantine, Release:
		return ui.DetailForm(ui.DetailField("Reason", v.dispositionReason))
	case Dispose:
		return ui.DetailForm(ui.DetailField("Quantity ("+string(v.state.Selected.Inventory.Amount.Unit())+")", v.amount), ui.DetailField("Reason", v.dispositionReason))
	case Tags:
		return ui.DetailForm(ui.DetailField("Tags", v.tags.Content))
	}
	return container.NewVBox()
}

func (v *View) detailFields(s State) framework.CanvasObject {
	if s.Selected == nil {
		return container.NewVBox()
	}
	r := s.Selected
	entry := ui.ReadonlyEntry
	form := ui.DetailForm(ui.DetailField("Ingredient", entry(r.Ingredient.Name)), ui.DetailField("Category", entry(string(r.Ingredient.Category))), ui.DetailField("On hand", entry(exactInventoryAmount(r.Inventory.Amount))), ui.DetailField("Reserved", entry(exactInventoryAmount(r.Inventory.ReservedAmount()))), ui.DetailField("Available", entry(exactInventoryAmount(r.Inventory.Available()))), ui.DetailField("Cost per "+string(r.Inventory.CostUnit), entry(r.Cost)), ui.DetailField("Status", entry(r.Status)), ui.DetailField("Disposition", entry(cmp.Or(string(r.Inventory.Status), "active"))), ui.DetailField("Disposition reason", entry(r.Inventory.Reason)), ui.DetailField("Revision", entry(fmt.Sprint(r.Inventory.Revision))), ui.DetailField("Inventory ID", entry(r.Inventory.ID.String())), ui.DetailField("Ingredient ID", entry(r.Inventory.IngredientID.String())), ui.DetailField("Tags", ui.TagPillsCSV(r.Inventory.Tags.Canonical().String())), ui.DetailField("Last updated", entry(formatInventoryTime(r.Inventory.LastUpdated))))
	reasons := container.NewVBox()
	disabledReasons := map[string]bool{}
	for _, id := range []actions.ID{inventory.ControlAdjust, inventory.ControlSet, inventory.ControlQuarantine, inventory.ControlRelease, inventory.ControlDispose} {
		state := s.Actions[id]
		if state.Visible && !state.Enabled && state.DisabledReason != "" && !disabledReasons[state.DisabledReason] {
			disabledReasons[state.DisabledReason] = true
			label := widget.NewLabel(state.DisabledReason)
			label.Wrapping = framework.TextWrapWord
			reasons.Add(label)
		}
	}
	return container.NewVBox(ui.ActionBar(nil, []framework.CanvasObject{v.adjust, v.set, v.tagAction}), ui.ActionBar(nil, []framework.CanvasObject{v.quarantine, v.release, v.dispose, v.historyAction}), form, reasons)
}

func (v *View) changed() {
	if v.rendering {
		return
	}
	s := v.presenter.Snapshot()
	if !isMutationMode(s.Mode) {
		return
	}
	v.readForm()
}
func (v *View) readForm() {
	s := v.presenter.Snapshot()
	f := Form{CostUnit: measurement.Unit(v.costUnit.Text), DispositionReason: v.dispositionReason.Text, Amount: v.amount.Text, Cost: v.cost.Text, Tags: v.tags.CSV(), Reason: inventorymodels.AdjustmentReason(v.reason.Selected), ReplaceTags: s.Mode != Tags}
	v.presenter.SetForm(f)
}
func (v *View) populate(f Form) {
	v.rendering = true
	defer func() { v.rendering = false }()
	v.costUnit.SetText(string(f.CostUnit))
	v.dispositionReason.SetText(f.DispositionReason)
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
	if isMutationMode(s.Mode) {
		if v.renderedMode != s.Mode || v.renderedInstance != s.FormInstance || !reflect.DeepEqual(v.renderedForm, s.Form) {
			v.populate(s.Form)
			v.renderedMode = s.Mode
			v.renderedInstance = s.FormInstance
			v.renderedForm = s.Form
			name := "Inventory item"
			if s.Selected != nil {
				name = s.Selected.Ingredient.Name
			}
			v.mutation = ui.StandardFormPage(ui.FormPage{Title: mutationTitle(s.Mode, name), Breadcrumb: v.breadcrumb(name), Fields: v.mutationFields(s.Mode), Status: v.formStatus, Save: v.save, Cancel: v.cancel}).(*framework.Container)
		}
	}
	if s.Selected != nil {
		v.title.SetText(s.Selected.Ingredient.Name)
		v.crumb.SetText(s.Selected.Ingredient.Name)
		if s.Mode == Viewing {
			v.detail = ui.StandardFormPage(ui.FormPage{Title: s.Selected.Ingredient.Name, Breadcrumb: v.breadcrumb(s.Selected.Ingredient.Name), Fields: v.detailFields(s), Status: v.formStatus}).(*framework.Container)
		}
	}
	for _, item := range []struct {
		button *ui.SemanticButton
		id     actions.ID
	}{{v.adjust, inventory.ControlAdjust}, {v.set, inventory.ControlSet}, {v.tagAction, inventory.ControlTags}, {v.quarantine, inventory.ControlQuarantine}, {v.release, inventory.ControlRelease}, {v.dispose, inventory.ControlDispose}, {v.historyAction, inventory.ControlHistory}} {
		item.button.Hidden = s.Selected == nil || !actionVisible(s.Actions, item.id)
		if s.Submitting || !actionEnabled(s.Actions, item.id) {
			item.button.Disable()
		} else {
			item.button.Enable()
		}
	}
	v.adjust.Hidden = s.Selected == nil || !actionVisible(s.Actions, inventory.ControlAdjust)
	v.set.Hidden = s.Selected == nil || !actionVisible(s.Actions, inventory.ControlSet)
	v.tagAction.Hidden = s.Selected == nil || !actionVisible(s.Actions, inventory.ControlTags)
	v.refresh.Hidden = !actionVisible(s.Actions, inventory.ControlList)
	v.create.Hidden = !actionVisible(s.Actions, inventory.ControlCreate)
	if s.Submitting || !s.Dirty {
		v.save.Disable()
	} else {
		v.save.Enable()
		v.cancel.Enable()
	}
	if s.Submitting {
		v.cancel.Disable()
	} else {
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
	case SelectingIngredient:
		options := make([]string, len(s.Candidates))
		for i, candidate := range s.Candidates {
			options[i] = candidate.Name + " (" + string(candidate.Unit) + ")"
		}
		v.ingredient = widget.NewSelect(options, func(label string) {
			for i, option := range options {
				if option == label {
					v.presenter.SelectIngredient(s.Candidates[i].ID)
					return
				}
			}
		})
		fields := container.NewVBox(widget.NewLabel("Choose an active ingredient without stock."), v.ingredient)
		if s.CandidateStatus == ui.Loading {
			fields.Add(widget.NewLabel("Loading ingredients…"))
		} else if len(options) == 0 && s.Err == nil {
			fields.Add(widget.NewLabel("All active ingredients already have stock. Create an ingredient first."))
		}
		v.root.Objects = []framework.CanvasObject{ui.StandardFormPage(ui.FormPage{Title: "Receive new stock", Breadcrumb: v.breadcrumb("Choose ingredient"), Fields: fields, Status: v.formStatus, Cancel: v.cancel})}
	case MovementHistory:
		v.root.Objects = []framework.CanvasObject{ui.StandardFormPage(ui.FormPage{Title: "Stock movement history", Breadcrumb: v.breadcrumb(s.Selected.Ingredient.Name), Fields: movementFields(s.Movements), Status: v.formStatus})}
	case Adjust, Set, Tags, Quarantine, Release, Dispose:
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

func movementFields(movements []inventorymodels.Movement) framework.CanvasObject {
	fields := container.NewVBox()
	if len(movements) == 0 {
		fields.Add(widget.NewLabel("No stock movements recorded"))
	}
	for _, movement := range movements {
		fields.Add(ui.DetailForm(ui.DetailField("Recorded", ui.ReadonlyEntry(formatInventoryTime(movement.At))), ui.DetailField("Quantity", ui.ReadonlyEntry(fmt.Sprintf("Before: %g %s; After: %g %s", movement.Before, movement.Unit, movement.After, movement.Unit))), ui.DetailField("Disposition", ui.ReadonlyEntry("Before: "+string(movement.BeforeStatus)+"; After: "+string(movement.AfterStatus))), ui.DetailField("Reason", ui.ReadonlyEntry(movement.Reason)), ui.DetailField("Movement ID", ui.ReadonlyEntry(movement.ID))))
		fields.Add(widget.NewSeparator())
	}
	return fields
}

func mutationTitle(mode Mode, name string) string {
	switch mode {
	case Quarantine:
		return "Quarantine: " + name
	case Release:
		return "Release quarantine: " + name
	case Dispose:
		return "Dispose stock: " + name
	case Browse, Viewing, Adjust, Set, Tags, MovementHistory, SelectingIngredient:
		return name
	}
	return name
}

func exactInventoryAmount(amount measurement.Amount) string {
	if amount == nil {
		return "N/A"
	}
	return fmt.Sprintf("%g %s", amount.Value(), amount.Unit())
}
