// Package fyne provides the bespoke Fyne presentation for inventory.
package fyne

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventory "github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	fyneui "github.com/TheFellow/go-modular-monolith/pkg/fyne"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
)

const LowStockThreshold = inventory.DefaultLowStockThreshold

type Mode uint8

const (
	Browse Mode = iota
	Adjust
	Set
	Tags
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
}

type Form struct {
	Amount, Cost, Tags string
	Reason             inventorymodels.AdjustmentReason
	ReplaceTags        bool
}

type State struct {
	Status       fyneui.LoadStatus
	Rows         []Row
	Selected     *Row
	Expression   string
	Stock        StockMode
	LowStock     float64
	Limit        int
	Cursor, Next paging.Cursor
	History      []paging.Cursor
	Mode         Mode
	Form         Form
	Err          error
	Submitting   bool
}

type loadResult struct {
	rows []Row
	next paging.Cursor
}

type Presenter struct {
	app     *app.Session
	dialogs fyneui.Dialogs
	load    *fyneui.LatestRequest[loadResult]
	submit  *fyneui.Submission
	mu      sync.Mutex
	state   State
	changed func(State)
}

func NewPresenter(session *app.Session, executor fyneui.Executor, dispatcher fyneui.Dispatcher, dialogs ...fyneui.Dialogs) *Presenter {
	p := &Presenter{app: session, state: State{Limit: paging.DefaultLimit, LowStock: LowStockThreshold}}
	if len(dialogs) > 0 {
		p.dialogs = dialogs[0]
	}
	p.load = fyneui.NewLatestRequest[loadResult](executor, dispatcher)
	p.submit = fyneui.NewSubmission(executor, dispatcher)
	return p
}

func (p *Presenter) OnChange(changed func(State)) { p.mu.Lock(); p.changed = changed; p.mu.Unlock() }
func (p *Presenter) Snapshot() State              { p.mu.Lock(); defer p.mu.Unlock(); return cloneState(p.state) }

func (p *Presenter) Load() {
	p.mu.Lock()
	req := inventory.ListRequest{Filter: p.state.Expression, Cursor: p.state.Cursor, Limit: p.state.Limit}
	threshold := p.state.LowStock
	if p.state.Stock == LowStock {
		req.LowStock = optional.Some(threshold)
	}
	p.mu.Unlock()
	p.load.Load(func() (loadResult, error) {
		page, err := p.app.Inventory.List(p.app.Context(), req)
		if err != nil {
			return loadResult{}, err
		}
		rows := make([]Row, 0, len(page.Items))
		for i, item := range page.Items {
			if item == nil {
				return loadResult{}, apperrors.Internalf("inventory %d missing", i)
			}
			ingredient, err := p.app.Ingredients.Get(p.app.Context(), item.IngredientID)
			if err != nil {
				return loadResult{}, fmt.Errorf("load ingredient %s: %w", item.IngredientID, err)
			}
			if ingredient == nil {
				return loadResult{}, apperrors.Internalf("ingredient %s missing", item.IngredientID)
			}
			rows = append(rows, makeRow(*item, *ingredient, threshold))
		}
		return loadResult{rows: rows, next: page.Next}, nil
	}, func(result fyneui.LoadState[loadResult]) {
		p.mu.Lock()
		p.state.Status, p.state.Err = result.Status, fyneui.PresentError(result.Err)
		if result.Status == fyneui.Loaded {
			selected := selectedID(p.state.Selected)
			p.state.Rows, p.state.Next = result.Value.rows, result.Value.next
			p.state.Selected = findRow(p.state.Rows, selected)
			if p.state.Selected == nil && len(p.state.Rows) > 0 {
				value := p.state.Rows[0]
				p.state.Selected = &value
			}
		}
		p.publishLocked()
		p.mu.Unlock()
		if result.Status == fyneui.Failed {
			fyneui.ShowPresentation(p.dialogs, result.Err)
		}
	})
}

func (p *Presenter) Filter(stock StockMode, expression string, lowStock float64, limit int) bool {
	if limit <= 0 {
		limit = paging.DefaultLimit
	}
	p.mu.Lock()
	if lowStock < 0 {
		p.state.Err = fyneui.PresentError(apperrors.Invalidf("low-stock threshold must be >= 0"))
		p.publishLocked()
		p.mu.Unlock()
		return false
	}
	p.state.Stock, p.state.Expression, p.state.LowStock, p.state.Limit = stock, strings.TrimSpace(expression), lowStock, limit
	p.state.Cursor, p.state.Next, p.state.History = "", "", nil
	p.mu.Unlock()
	p.Load()
	return true
}
func (p *Presenter) NextPage() {
	p.mu.Lock()
	if p.state.Next == "" {
		p.mu.Unlock()
		return
	}
	p.state.History = append(p.state.History, p.state.Cursor)
	p.state.Cursor = p.state.Next
	p.mu.Unlock()
	p.Load()
}
func (p *Presenter) PreviousPage() {
	p.mu.Lock()
	if len(p.state.History) == 0 {
		p.mu.Unlock()
		return
	}
	n := len(p.state.History) - 1
	p.state.Cursor = p.state.History[n]
	p.state.History = p.state.History[:n]
	p.mu.Unlock()
	p.Load()
}
func (p *Presenter) Select(id entity.InventoryID) {
	p.mu.Lock()
	if p.state.Mode != Browse || p.state.Submitting {
		p.mu.Unlock()
		return
	}
	p.state.Selected = findRow(p.state.Rows, id)
	p.publishLocked()
	p.mu.Unlock()
}

func (p *Presenter) StartAdjust() { p.start(Adjust) }
func (p *Presenter) StartSet()    { p.start(Set) }
func (p *Presenter) StartTags()   { p.start(Tags) }
func (p *Presenter) start(mode Mode) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state.Selected == nil {
		return
	}
	p.state.Mode, p.state.Err = mode, nil
	p.state.Form = Form{Tags: p.state.Selected.Inventory.Tags.Canonical().String(), ReplaceTags: mode != Tags}
	switch mode { //nolint:exhaustive // browse and adjust do not create inventory.
	case Set:
		p.state.Form.Amount = fmt.Sprintf("%.2f", p.state.Selected.Inventory.Amount.Value())
		if price, ok := p.state.Selected.Inventory.CostPerUnit.Unwrap(); ok {
			cents, _ := price.Cents()
			p.state.Form.Cost = fmt.Sprintf("%.2f", float64(cents)/100)
		}
	case Tags:
		p.state.Form.Tags = p.state.Selected.Inventory.Tags.Canonical().String()
	}
	p.publishLocked()
}
func (p *Presenter) Cancel() {
	p.mu.Lock()
	if !p.state.Submitting {
		p.state.Mode, p.state.Err = Browse, nil
	}
	p.publishLocked()
	p.mu.Unlock()
}

func (p *Presenter) Submit(form Form) bool {
	p.mu.Lock()
	mode, selected := p.state.Mode, p.state.Selected
	p.state.Form = form
	if selected == nil {
		p.state.Err = fyneui.PresentError(apperrors.Invalidf("inventory item is required"))
		p.publishLocked()
		p.mu.Unlock()
		return false
	}
	validated, err := validate(mode, form, selected.Ingredient.Unit, selected.Inventory.CostPerUnit)
	if err != nil {
		p.state.Err = fyneui.PresentError(err)
		p.publishLocked()
		p.mu.Unlock()
		return false
	}
	p.state.Err = nil
	p.state.Submitting = true
	p.publishLocked()
	p.mu.Unlock()
	accepted := p.submit.Submit(func() error {
		var desired *tag.Tags
		if form.ReplaceTags {
			desired = &validated.tags
		}
		switch mode { //nolint:exhaustive // browse mode does not submit a mutation.
		case Adjust:
			_, err = app.RunTaggedMutation(p.app.App, p.app.Context(), desired, func(ctx *middleware.Context) (*inventorymodels.Inventory, error) {
				return p.app.Inventory.Adjust(ctx, &inventorymodels.Patch{IngredientID: selected.Ingredient.ID, Reason: form.Reason, Delta: validated.amount, CostPerUnit: validated.cost})
			})
		case Set:
			amount, _ := validated.amount.Unwrap()
			cost, _ := validated.cost.Unwrap()
			_, err = app.RunTaggedMutation(p.app.App, p.app.Context(), desired, func(ctx *middleware.Context) (*inventorymodels.Inventory, error) {
				return p.app.Inventory.Set(ctx, &inventorymodels.Update{IngredientID: selected.Ingredient.ID, Amount: amount, CostPerUnit: cost})
			})
		case Tags:
			_, err = p.app.Tags.Replace(p.app.Context(), selected.Inventory.EntityUID(), validated.tags)
		default:
			err = apperrors.Invalidf("inventory form is not active")
		}
		return err
	}, func(err error) {
		p.mu.Lock()
		p.state.Submitting = false
		p.state.Err = fyneui.PresentError(err)
		if err == nil {
			p.state.Mode = Browse
		}
		p.publishLocked()
		p.mu.Unlock()
		fyneui.ShowPresentation(p.dialogs, err)
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
	if mode == Tags || form.ReplaceTags {
		tags, err := tag.ParseCollection(form.Tags)
		out.tags = tags
		if err != nil || mode == Tags {
			return out, err
		}
	}
	if mode == Adjust && !validReason(form.Reason) {
		return out, apperrors.Invalidf("reason is required")
	}
	amountText := strings.TrimSpace(form.Amount)
	if mode == Set && amountText == "" {
		return out, apperrors.Invalidf("amount is required")
	}
	if amountText != "" {
		value, err := parsePrecision2(amountText, "amount")
		if err != nil {
			return out, err
		}
		if mode == Set && value < 0 {
			return out, apperrors.Invalidf("quantity must be >= 0")
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
		return out, apperrors.Invalidf("at least one of amount or cost is required")
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
		return 0, apperrors.Invalidf("%s is required", name)
	}
	if dot := strings.IndexByte(raw, '.'); dot >= 0 && len(raw)-dot-1 > 2 {
		return 0, apperrors.Invalidf("%s must have at most 2 decimal places", name)
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, apperrors.Invalidf("invalid %s", name)
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
	return Row{Inventory: inv, Ingredient: ingredient, Quantity: fmt.Sprintf("%.2f %s", inv.Amount.Value(), inv.Amount.Unit()), Cost: cost, Status: StockStatus(inv.Amount, lowStock)}
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
	state.History = append([]paging.Cursor(nil), state.History...)
	if state.Selected != nil {
		v := *state.Selected
		state.Selected = &v
	}
	return state
}
func (p *Presenter) publishLocked() {
	if p.changed != nil {
		p.changed(cloneState(p.state))
	}
}
