// Package gui provides the bespoke GUI presentation for inventory.
package gui

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventory "github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	inventorypresentation "github.com/TheFellow/go-modular-monolith/app/domains/inventory/surfaces"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	toolkit "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

const LowStockThreshold = inventory.DefaultLowStockThreshold

type Mode uint8

const (
	Browse Mode = iota
	Viewing
	Adjust
	Set
	Tags
	Quarantine
	Release
	Dispose
	MovementHistory
	SelectingIngredient
)

type StockMode uint8

const (
	AllStock StockMode = iota
	LowStock
)

type Row struct {
	Inventory  inventorymodels.Inventory
	Ingredient models.Ingredient
	Quantity   string
	Cost       string
	Status     string
	CanAdjust  bool
	CanSet     bool
	CanTag     bool
	Actions    map[actions.ID]actions.State
}

type Form struct {
	CostUnit           measurement.Unit
	Amount, Cost, Tags string
	DispositionReason  string
	Reason             inventorymodels.AdjustmentReason
	ReplaceTags        bool
}

type State struct {
	Candidates      []models.Ingredient
	CandidateStatus toolkit.LoadStatus
	Creating        bool
	Status          toolkit.LoadStatus
	Rows            []Row
	Selected        *Row
	Expression      string
	Stock           StockMode
	LowStock        float64
	Limit           int
	Cursor, Next    paging.Cursor
	History         []paging.Cursor
	Movements       []inventorymodels.Movement
	Mode            Mode
	Form            Form
	Err             error
	Submitting      bool
	Dirty           bool
	CanAdjust       bool
	CanSet          bool
	CanTag          bool
	CanList         bool
	Actions         map[actions.ID]actions.State
	FormInstance    uint64
}

type loadResult struct {
	rows []Row
	next paging.Cursor
}

type Presenter struct {
	candidates *toolkit.LatestRequest[[]models.Ingredient]
	app        *app.Session
	dialogs    toolkit.Dialogs
	load       *toolkit.LatestRequest[loadResult]
	movements  *toolkit.LatestRequest[[]inventorymodels.Movement]
	submit     *toolkit.Submission
	mu         sync.Mutex
	state      State
	changed    func(State)
	projector  inventory.ActionProjector
	sort       toolkit.TableSort
}

func NewPresenter(session *app.Session, executor toolkit.Executor, dispatcher toolkit.Dispatcher, dialogs ...toolkit.Dialogs) *Presenter {
	projector := inventory.NewActionProjector()
	p := &Presenter{app: session, state: State{Limit: toolkit.PageLimit, LowStock: LowStockThreshold}, projector: projector}
	if len(dialogs) > 0 {
		p.dialogs = dialogs[0]
	}
	p.load = toolkit.NewLatestRequest[loadResult](executor, dispatcher)
	p.submit = toolkit.NewSubmission(executor, dispatcher)
	p.movements = toolkit.NewLatestRequest[[]inventorymodels.Movement](executor, dispatcher)
	p.candidates = toolkit.NewLatestRequest[[]models.Ingredient](executor, dispatcher)
	if err := p.permissionsLocked(); err != nil {
		p.state.Err = toolkit.PresentError(err)
		toolkit.ShowPresentation(p.dialogs, err)
	}
	return p
}

func (p *Presenter) OnChange(changed func(State)) { p.mu.Lock(); p.changed = changed; p.mu.Unlock() }
func (p *Presenter) Snapshot() State              { p.mu.Lock(); defer p.mu.Unlock(); return cloneState(p.state) }

func (p *Presenter) Load() {
	p.mu.Lock()
	if !actionEnabled(p.state.Actions, inventory.ControlList) {
		p.mu.Unlock()
		return
	}
	p.state.Cursor, p.state.Next, p.state.History = "", "", nil
	p.mu.Unlock()
	p.loadPage(false)
}

func (p *Presenter) loadPage(appendPage bool) {
	p.mu.Lock()
	req := inventory.ListRequest{Filter: p.state.Expression, Cursor: p.state.Cursor, Limit: p.state.Limit}
	threshold := p.state.LowStock
	if p.state.Stock == LowStock {
		req.LowStock = optional.Some(threshold)
	}
	p.mu.Unlock()
	p.load.LoadContext(p.app.Context(), func(ctx context.Context) (loadResult, error) {
		op := p.app.ContextFrom(ctx)
		page, err := p.app.Inventory.List(op, req)
		if err != nil {
			return loadResult{}, err
		}
		rows := make([]Row, 0, len(page.Items))
		for i, item := range page.Items {
			if item == nil {
				return loadResult{}, errors.Internalf("inventory %d missing", i)
			}
			ingredient, err := p.app.Ingredients.Get(op, item.IngredientID)
			if errors.IsNotFound(err) && item.Status != inventorymodels.StatusActive {
				ingredient = &models.Ingredient{ID: item.IngredientID, Name: item.IngredientName, Unit: item.Amount.Unit()}
				err = nil
			}
			if err != nil {
				return loadResult{}, fmt.Errorf("load ingredient %s: %w", item.IngredientID, err)
			}
			if ingredient == nil {
				return loadResult{}, errors.Internalf("ingredient %s missing", item.IngredientID)
			}
			row := makeRow(*item, *ingredient, threshold)
			states, err := p.projector.Project(op, op.Principal(), item)
			if err != nil {
				return loadResult{}, fmt.Errorf("project inventory actions: %w", err)
			}
			row.Actions = indexActions(states)
			row.CanAdjust = actionVisible(row.Actions, inventory.ControlAdjust)
			row.CanSet = actionVisible(row.Actions, inventory.ControlSet)
			row.CanTag = actionVisible(row.Actions, inventory.ControlTags)
			rows = append(rows, row)
		}
		return loadResult{rows: rows, next: page.Next}, nil
	}, func(result toolkit.LoadState[loadResult]) {
		p.mu.Lock()
		p.state.Status, p.state.Err = result.Status, toolkit.PresentError(result.Err)
		if result.Status == toolkit.Loaded {
			selected := selectedID(p.state.Selected)
			if appendPage {
				p.state.Rows = append(p.state.Rows, result.Value.rows...)
			} else {
				p.state.Rows = result.Value.rows
			}
			p.sortRowsLocked()
			p.state.Next = result.Value.next
			// A form owns the target and revision captured when it opened. A list
			// refresh may finish later, including while receiving stock whose row
			// does not exist yet, but must never replace that command target.
			if p.state.Mode == Browse || p.state.Mode == Viewing {
				p.state.Selected = findRow(p.state.Rows, selected)
				// Keep a latent selection for command compatibility; Browse still renders
				// only the collection until the actor explicitly selects a table row.
				if p.state.Selected == nil && len(p.state.Rows) > 0 {
					value := p.state.Rows[0]
					p.state.Selected = &value
				}
			}
			if err := p.permissionsLocked(); err != nil {
				p.state.Err = toolkit.PresentError(err)
			}
		}
		p.publishLocked()
		p.mu.Unlock()
		if result.Status == toolkit.Failed {
			toolkit.ShowPresentation(p.dialogs, result.Err)
		}
	})
}

func (p *Presenter) SortRows(column int, direction toolkit.SortDirection) {
	if column < 0 || column > 9 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sort = toolkit.TableSort{Column: column, Direction: direction}
	p.sortRowsLocked()
	p.publishLocked()
}

func (p *Presenter) sortRowsLocked() {
	toolkit.ApplyTableSort(p.state.Rows, p.sort, func(column int, left, right Row) int {
		switch column {
		case 0:
			return cmp.Compare(left.Ingredient.Name, right.Ingredient.Name)
		case 1:
			return cmp.Compare(left.Inventory.IngredientID.String(), right.Inventory.IngredientID.String())
		case 2:
			return cmp.Compare(left.Inventory.Amount.Value(), right.Inventory.Amount.Value())
		case 3:
			return cmp.Compare(left.Inventory.ReservedAmount().Value(), right.Inventory.ReservedAmount().Value())
		case 4:
			return cmp.Compare(left.Inventory.Available().Value(), right.Inventory.Available().Value())
		case 5:
			return cmp.Compare(left.Inventory.Amount.Unit(), right.Inventory.Amount.Unit())
		case 6:
			leftCost, leftOK := left.Inventory.CostPerUnit.Unwrap()
			rightCost, rightOK := right.Inventory.CostPerUnit.Unwrap()
			if leftOK != rightOK {
				if leftOK {
					return 1
				}
				return -1
			}
			if result := cmp.Compare(leftCost.Currency.Code, rightCost.Currency.Code); result != 0 {
				return result
			}
			return leftCost.Amount.Cmp(rightCost.Amount)
		case 7:
			return left.Inventory.LastUpdated.Compare(right.Inventory.LastUpdated)
		case 8:
			return cmp.Compare(left.Inventory.Tags.Canonical().String(), right.Inventory.Tags.Canonical().String())
		case 9:
			return cmp.Compare(left.Status, right.Status)
		}
		return 0
	})
}

func (p *Presenter) Filter(stock StockMode, expression string, lowStock float64, limit int) bool {
	if limit <= 0 {
		limit = toolkit.PageLimit
	}
	p.mu.Lock()
	if !actionEnabled(p.state.Actions, inventory.ControlList) {
		p.mu.Unlock()
		return false
	}
	if lowStock < 0 {
		p.state.Err = toolkit.PresentError(errors.Invalidf("low-stock threshold must be >= 0"))
		p.publishLocked()
		p.mu.Unlock()
		return false
	}
	p.state.Stock, p.state.Expression, p.state.LowStock, p.state.Limit = stock, strings.TrimSpace(expression), lowStock, limit
	p.state.Cursor, p.state.Next, p.state.History = "", "", nil
	p.mu.Unlock()
	p.loadPage(false)
	return true
}
func (p *Presenter) NextPage() {
	p.mu.Lock()
	if p.state.Next == "" || !actionEnabled(p.state.Actions, inventory.ControlList) {
		p.mu.Unlock()
		return
	}
	p.state.History = append(p.state.History, p.state.Cursor)
	p.state.Cursor = p.state.Next
	p.mu.Unlock()
	p.loadPage(true)
}
func (p *Presenter) PreviousPage() {
	p.mu.Lock()
	if len(p.state.History) == 0 || !actionEnabled(p.state.Actions, inventory.ControlList) {
		p.mu.Unlock()
		return
	}
	n := len(p.state.History) - 1
	p.state.Cursor = p.state.History[n]
	p.state.History = p.state.History[:n]
	p.mu.Unlock()
	p.loadPage(false)
}
func (p *Presenter) Select(id entity.InventoryID) {
	p.mu.Lock()
	if p.state.Mode != Browse || p.state.Submitting {
		p.mu.Unlock()
		return
	}
	p.state.Selected = findRow(p.state.Rows, id)
	if p.state.Selected != nil {
		p.state.Mode, p.state.Dirty, p.state.Err = Viewing, false, nil
		p.state.FormInstance++
		if err := p.permissionsLocked(); err != nil {
			p.state.Err = toolkit.PresentError(err)
		}
	}
	p.publishLocked()
	p.mu.Unlock()
}

// Back returns to the exact filtered and paged list state used to open detail.
func (p *Presenter) Back() {
	p.mu.Lock()
	if p.state.Mode == MovementHistory {
		p.state.Mode, p.state.Err = Viewing, nil
		p.publishLocked()
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()
	p.leaveDetail(false)
}

// ResetList returns to an unfiltered first page for navigation and breadcrumbs.
func (p *Presenter) ResetList() { p.leaveDetail(true) }

func (p *Presenter) leaveDetail(reset bool) {
	p.mu.Lock()
	if p.state.Submitting {
		p.mu.Unlock()
		return
	}
	dirty := p.state.Dirty
	p.mu.Unlock()
	proceed := func() {
		p.mu.Lock()
		if reset {
			p.state.Expression, p.state.Stock, p.state.LowStock, p.state.Limit = "", AllStock, LowStockThreshold, toolkit.PageLimit
			p.state.Cursor, p.state.Next, p.state.History = "", "", nil
		}
		p.state.Mode, p.state.Dirty, p.state.Err = Browse, false, nil
		p.state.Creating = false
		p.publishLocked()
		p.mu.Unlock()
		if reset {
			p.Load()
		}
	}
	if dirty {
		if p.dialogs == nil {
			return
		}
		p.dialogs.Confirm("Discard changes?", "Discard unsaved inventory changes?", func(ok bool) {
			if ok {
				proceed()
			}
		})
		return
	}
	proceed()
}

func (p *Presenter) permissionsLocked() error {
	p.state.Actions = nil
	p.state.CanList, p.state.CanAdjust, p.state.CanSet, p.state.CanTag = false, false, false, false
	var selected *inventorymodels.Inventory
	if p.state.Selected != nil {
		selected = &p.state.Selected.Inventory
	}
	states, err := p.projector.Project(p.app.Context(), p.app.Context().Principal(), selected)
	if err != nil {
		return err
	}
	p.state.Actions = indexActions(states)
	p.state.CanList = actionVisible(p.state.Actions, inventory.ControlList)
	p.state.CanAdjust = actionVisible(p.state.Actions, inventory.ControlAdjust)
	p.state.CanSet = actionVisible(p.state.Actions, inventory.ControlSet)
	p.state.CanTag = actionVisible(p.state.Actions, inventory.ControlTags)
	return nil
}

func (p *Presenter) StartQuarantine() { p.start(Quarantine) }
func (p *Presenter) StartRelease()    { p.start(Release) }
func (p *Presenter) StartDispose()    { p.start(Dispose) }
func (p *Presenter) StartAdjust()     { p.start(Adjust) }
func (p *Presenter) StartSet()        { p.start(Set) }
func (p *Presenter) StartTags()       { p.start(Tags) }
func (p *Presenter) start(mode Mode) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state.Selected == nil {
		return
	}
	if err := p.permissionsLocked(); err != nil {
		p.state.Err = toolkit.PresentError(err)
		p.publishLocked()
		return
	}
	allowed := actionEnabled(p.state.Actions, controlForMode(mode)) && !p.state.Submitting
	if !allowed {
		return
	}
	p.state.Mode, p.state.Err, p.state.Dirty = mode, nil, p.state.Creating
	p.state.FormInstance++
	p.state.Form = Form{Tags: p.state.Selected.Inventory.Tags.Canonical().String(), ReplaceTags: mode != Tags, CostUnit: p.state.Selected.Inventory.CostUnit}
	switch mode {
	case Set:
		p.state.Form.Amount = strconv.FormatFloat(p.state.Selected.Inventory.Amount.Value(), 'f', -1, 64)
		if price, ok := p.state.Selected.Inventory.CostPerUnit.Unwrap(); ok {
			cents, _ := price.Cents()
			p.state.Form.Cost = fmt.Sprintf("%.2f", float64(cents)/100)
		}
	case Tags:
		p.state.Form.Tags = p.state.Selected.Inventory.Tags.Canonical().String()
	case Browse, Viewing, Adjust, Quarantine, Release, Dispose, MovementHistory, SelectingIngredient:
	}
	p.publishLocked()
}
func (p *Presenter) Cancel() {
	p.mu.Lock()
	if !p.state.Submitting {
		if isMutationMode(p.state.Mode) || p.state.Mode == SelectingIngredient {
			mode := p.state.Mode
			p.state.Mode = Viewing
			if p.state.Creating {
				p.state.Mode, p.state.Creating, p.state.Selected = Browse, false, nil
			}
			p.state.Form = Form{}
			p.state.Dirty, p.state.Err = false, nil
			p.state.FormInstance++
			_ = mode
		}
	}
	p.publishLocked()
	p.mu.Unlock()
}

func (p *Presenter) SetForm(form Form) {
	p.mu.Lock()
	if !isMutationMode(p.state.Mode) {
		p.mu.Unlock()
		return
	}
	baseline := p.formForModeLocked(p.state.Mode)
	p.state.Form, p.state.Dirty = form, p.state.Creating || !reflect.DeepEqual(form, baseline)
	p.publishLocked()
	p.mu.Unlock()
}

func (p *Presenter) formForModeLocked(mode Mode) Form {
	if p.state.Selected == nil {
		return Form{}
	}
	f := Form{Tags: p.state.Selected.Inventory.Tags.Canonical().String(), ReplaceTags: mode != Tags, CostUnit: p.state.Selected.Inventory.CostUnit}
	if mode == Set {
		f.Amount = strconv.FormatFloat(p.state.Selected.Inventory.Amount.Value(), 'f', -1, 64)
		if price, ok := p.state.Selected.Inventory.CostPerUnit.Unwrap(); ok {
			cents, _ := price.Cents()
			f.Cost = fmt.Sprintf("%.2f", float64(cents)/100)
		}
	}
	return f
}

func (p *Presenter) Submit(form Form) bool {
	p.mu.Lock()
	mode, selected := p.state.Mode, p.state.Selected
	p.state.Form = form
	if selected == nil {
		p.state.Err = toolkit.PresentError(errors.Invalidf("inventory item is required"))
		p.publishLocked()
		p.mu.Unlock()
		return false
	}
	if p.state.Submitting || !isMutationMode(mode) {
		p.mu.Unlock()
		return false
	}
	if err := p.permissionsLocked(); err != nil {
		p.state.Err = toolkit.PresentError(err)
		p.publishLocked()
		p.mu.Unlock()
		return false
	}
	if !actionEnabled(p.state.Actions, controlForMode(mode)) {
		p.mu.Unlock()
		return false
	}
	validated, err := validate(mode, form, selected.Inventory.Amount.Unit(), selected.Inventory.CostPerUnit)
	if err != nil {
		p.state.Err = toolkit.PresentError(err)
		p.publishLocked()
		p.mu.Unlock()
		return false
	}
	p.state.Err = nil
	p.state.Submitting = true
	p.publishLocked()
	p.mu.Unlock()
	var savedStock *inventorymodels.Inventory
	accepted := p.submit.Submit(func() error {
		var desired *tag.Tags
		if form.ReplaceTags {
			desired = &validated.tags
		}
		switch mode {
		case Adjust:
			_, err = p.app.Inventory.Adjust(p.app.Context(), &inventorymodels.Patch{CostUnit: cmp.Or(form.CostUnit, selected.Inventory.CostUnit), Revision: selected.Inventory.Revision, IngredientID: selected.Ingredient.ID, Reason: form.Reason, Delta: validated.amount, CostPerUnit: validated.cost}, tag.Replace(desired, selected.Inventory.Tags))
		case Set:
			amount, _ := validated.amount.Unwrap()
			cost, _ := validated.cost.Unwrap()
			savedStock, err = p.app.Inventory.Set(p.app.Context(), &inventorymodels.Update{CostUnit: cmp.Or(form.CostUnit, selected.Inventory.CostUnit), Revision: selected.Inventory.Revision, IngredientID: selected.Ingredient.ID, Amount: amount, CostPerUnit: cost}, tag.Replace(desired, selected.Inventory.Tags))
		case Quarantine, Release:
			_, err = p.app.Inventory.Disposition(p.app.Context(), inventorymodels.Disposition{IngredientID: selected.Inventory.IngredientID, Revision: selected.Inventory.Revision, Quarantine: mode == Quarantine, Reason: form.DispositionReason})
		case Dispose:
			amount, _ := validated.amount.Unwrap()
			_, err = p.app.Inventory.Dispose(p.app.Context(), inventorymodels.Disposal{IngredientID: selected.Inventory.IngredientID, Revision: selected.Inventory.Revision, Amount: amount, Reason: form.DispositionReason})
		case Tags:
			_, err = p.app.Tags.Replace(p.app.Context(), selected.Inventory.EntityUID(), validated.tags, selected.Inventory.Tags)
		case Browse, Viewing, MovementHistory, SelectingIngredient:
			err = errors.Invalidf("inventory form is not active")
		}
		return err
	}, func(err error) {
		p.mu.Lock()
		p.state.Submitting = false
		p.state.Err = toolkit.PresentError(err)
		if err == nil {
			p.state.Mode, p.state.Dirty, p.state.Creating = Viewing, false, false
			if savedStock != nil {
				row := makeRow(*savedStock, selected.Ingredient, p.state.LowStock)
				p.state.Selected = &row
			}
		}
		p.publishLocked()
		p.mu.Unlock()
		toolkit.ShowPresentation(p.dialogs, err)
		if err == nil {
			p.Load()
		}
	})
	if !accepted {
		p.mu.Lock()
		p.state.Submitting = p.submit.Active()
		p.publishLocked()
		p.mu.Unlock()
	}
	return accepted
}

type validatedForm struct {
	amount optional.Value[measurement.Amount]
	cost   optional.Value[money.Price]
	tags   tag.Tags
}

func validate(mode Mode, form Form, unit measurement.Unit, existingCost optional.Value[money.Price]) (validatedForm, error) {
	var out validatedForm
	if mode == Quarantine || mode == Release || mode == Dispose {
		if strings.TrimSpace(form.DispositionReason) == "" {
			return out, errors.Invalidf("a disposition reason is required")
		}
		if mode != Dispose {
			return out, nil
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(form.Amount), 64)
		if err != nil {
			return out, errors.Invalidf("invalid quantity")
		}
		if value <= 0 || math.IsInf(value, 0) || math.IsNaN(value) {
			return out, errors.Invalidf("positive disposal amount is required")
		}
		amount, err := measurement.NewAmount(value, unit)
		out.amount = optional.Some(amount)
		return out, err
	}
	if mode == Tags || form.ReplaceTags {
		tags, err := tag.ParseCollection(form.Tags)
		out.tags = tags
		if err != nil || mode == Tags {
			return out, err
		}
	}
	if mode == Adjust && !validReason(form.Reason) {
		return out, errors.Invalidf("reason is required")
	}
	amountText := strings.TrimSpace(form.Amount)
	if mode == Set && amountText == "" {
		return out, errors.Invalidf("amount is required")
	}
	if amountText != "" {
		value, err := strconv.ParseFloat(amountText, 64)
		if err != nil {
			return out, errors.Invalidf("invalid amount")
		}
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return out, errors.Invalidf("amount must be finite")
		}
		if mode == Set && value < 0 {
			return out, errors.Invalidf("quantity must be >= 0")
		}
		amount, err := measurement.NewAmount(value, unit)
		if err != nil {
			return out, err
		}
		out.amount = optional.Some(amount)
	}
	cost := strings.TrimSpace(form.Cost)
	if mode == Set && cost == "" {
		if price, ok := existingCost.Unwrap(); ok {
			out.cost = optional.Some(price)
			return out, nil
		}
		cost = "0.00"
	}
	if cost != "" {
		price, err := parseInventoryPrice(cost, existingCost)
		if err != nil {
			return out, err
		}
		out.cost = optional.Some(price)
	}
	if mode == Adjust && out.amount.IsNone() && out.cost.IsNone() {
		return out, errors.Invalidf("at least one of amount or cost is required")
	}
	return out, nil
}

func parseInventoryPrice(raw string, existing optional.Value[money.Price]) (money.Price, error) {
	if strings.HasPrefix(strings.TrimSpace(raw), "$") || len(strings.Fields(raw)) == 2 {
		return money.ParsePrice(raw)
	}
	if _, err := parsePrecision2(raw, "cost"); err != nil {
		return money.Price{}, err
	}
	curr := currency.USD
	if price, ok := existing.Unwrap(); ok {
		curr = price.Currency
	}
	return money.NewPrice(raw, curr)
}
func parsePrecision2(raw, name string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, errors.Invalidf("%s is required", name)
	}
	if dot := strings.IndexByte(raw, '.'); dot >= 0 && len(raw)-dot-1 > 2 {
		return 0, errors.Invalidf("%s must have at most 2 decimal places", name)
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, errors.Invalidf("invalid %s", name)
	}
	return value, nil
}
func validReason(r inventorymodels.AdjustmentReason) bool {
	switch r {
	case inventorymodels.ReasonReceived, inventorymodels.ReasonUsed, inventorymodels.ReasonSpilled, inventorymodels.ReasonExpired, inventorymodels.ReasonCorrected:
		return true
	}
	return false
}
func makeRow(inv inventorymodels.Inventory, ingredient models.Ingredient, lowStock float64) Row {
	cost := "N/A"
	if value, ok := inv.CostPerUnit.Unwrap(); ok {
		cost = value.String()
	}
	status := StockStatus(inv.Available(), lowStock)
	if inv.Status != "" && inv.Status != inventorymodels.StatusActive {
		status = strings.ToUpper(string(inv.Status))
	}
	return Row{Inventory: inv, Ingredient: ingredient, Quantity: fmt.Sprintf("%.2f %s", inv.Amount.Value(), inv.Amount.Unit()), Cost: cost, Status: status}
}
func StockStatus(amount measurement.Amount, lowStock float64) string {
	if amount == nil || amount.Value() <= 0 {
		return "OUT"
	}
	if amount.Value() <= lowStock {
		return "LOW"
	}
	return "OK"
}
func selectedID(row *Row) entity.InventoryID {
	if row == nil {
		return entity.InventoryID{}
	}
	return row.Inventory.ID
}
func findRow(rows []Row, id entity.InventoryID) *Row {
	for i := range rows {
		if rows[i].Inventory.ID == id {
			v := rows[i]
			return &v
		}
	}
	return nil
}
func cloneState(state State) State {
	state.Rows = append([]Row(nil), state.Rows...)
	state.Candidates = append([]models.Ingredient(nil), state.Candidates...)
	for i := range state.Rows {
		state.Rows[i].Actions = cloneActions(state.Rows[i].Actions)
	}
	state.History = append([]paging.Cursor(nil), state.History...)
	state.Movements = append([]inventorymodels.Movement(nil), state.Movements...)
	state.Actions = cloneActions(state.Actions)
	if state.Selected != nil {
		v := *state.Selected
		v.Actions = cloneActions(v.Actions)
		state.Selected = &v
	}
	return state
}
func indexActions(states []actions.State) map[actions.ID]actions.State {
	out := make(map[actions.ID]actions.State, len(states))
	for _, state := range states {
		out[state.ID] = state
	}
	return out
}
func actionVisible(states map[actions.ID]actions.State, id actions.ID) bool {
	return states[id].Visible
}
func actionEnabled(states map[actions.ID]actions.State, id actions.ID) bool {
	state, ok := states[id]
	return ok && state.Visible && state.Enabled
}
func cloneActions(in map[actions.ID]actions.State) map[actions.ID]actions.State {
	out := make(map[actions.ID]actions.State, len(in))
	maps.Copy(out, in)
	return out
}
func (p *Presenter) publishLocked() {
	if p.changed != nil {
		p.changed(cloneState(p.state))
	}
}

func isMutationMode(mode Mode) bool {
	return mode == Adjust || mode == Set || mode == Tags || mode == Quarantine || mode == Release || mode == Dispose
}
func controlForMode(mode Mode) actions.ID {
	return map[Mode]actions.ID{Adjust: inventory.ControlAdjust, Set: inventory.ControlSet, Tags: inventory.ControlTags, Quarantine: inventory.ControlQuarantine, Release: inventory.ControlRelease, Dispose: inventory.ControlDispose}[mode]
}

func (p *Presenter) ShowHistory() {
	p.mu.Lock()
	if p.state.Selected == nil || p.state.Submitting || p.state.Dirty {
		p.mu.Unlock()
		return
	}
	if err := p.permissionsLocked(); err != nil {
		p.state.Err = toolkit.PresentError(err)
		p.publishLocked()
		p.mu.Unlock()
		return
	}
	if !actionEnabled(p.state.Actions, inventory.ControlHistory) {
		p.mu.Unlock()
		return
	}
	id := p.state.Selected.Inventory.IngredientID
	p.state.Mode, p.state.Movements, p.state.Err = MovementHistory, nil, nil
	p.publishLocked()
	p.mu.Unlock()
	p.movements.LoadContext(p.app.Context(), func(ctx context.Context) ([]inventorymodels.Movement, error) {
		return p.app.Inventory.History(p.app.ContextFrom(ctx), id)
	}, func(result toolkit.LoadState[[]inventorymodels.Movement]) {
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.state.Mode != MovementHistory || p.state.Selected == nil || p.state.Selected.Inventory.IngredientID != id {
			return
		}
		p.state.Err = toolkit.PresentError(result.Err)
		if result.Status == toolkit.Loaded {
			p.state.Movements = result.Value
		}
		p.publishLocked()
	})
}

func (p *Presenter) StartNew() {
	p.mu.Lock()
	if p.state.Mode != Browse || p.state.Submitting || !actionEnabled(p.state.Actions, inventory.ControlCreate) {
		p.mu.Unlock()
		return
	}
	p.state.Mode, p.state.Creating, p.state.Err, p.state.Candidates = SelectingIngredient, true, nil, nil
	p.publishLocked()
	p.mu.Unlock()
	p.candidates.LoadContext(p.app.Context(), func(ctx context.Context) ([]models.Ingredient, error) {
		return inventorypresentation.StockingCandidates(app.NewSession(p.app.ContextFrom(ctx), p.app.App), p.projector)
	}, func(result toolkit.LoadState[[]models.Ingredient]) {
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.state.Mode != SelectingIngredient {
			return
		}
		p.state.CandidateStatus, p.state.Err = result.Status, toolkit.PresentError(result.Err)
		if result.Status == toolkit.Loaded {
			p.state.Candidates = result.Value
		}
		p.publishLocked()
	})
}

func (p *Presenter) SelectIngredient(id entity.IngredientID) {
	p.mu.Lock()
	if p.state.Mode != SelectingIngredient || p.state.Submitting {
		p.mu.Unlock()
		return
	}
	for _, ingredient := range p.state.Candidates {
		if ingredient.ID != id {
			continue
		}
		stock := inventorymodels.Inventory{IngredientID: id, IngredientName: ingredient.Name, Status: inventorymodels.StatusActive, Amount: measurement.MustAmount(0, ingredient.Unit), CostUnit: ingredient.Unit}
		row := makeRow(stock, ingredient, p.state.LowStock)
		p.state.Selected = &row
		p.mu.Unlock()
		p.start(Set)
		return
	}
	p.mu.Unlock()
}
