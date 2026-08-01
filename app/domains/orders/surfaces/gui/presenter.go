// Package gui provides the bespoke retained-mode desktop presentation for orders.
package gui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/TheFellow/go-modular-monolith/app"
	menus "github.com/TheFellow/go-modular-monolith/app/domains/menus"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	orders "github.com/TheFellow/go-modular-monolith/app/domains/orders"
	ordersauthz "github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	pkgAuthz "github.com/TheFellow/go-modular-monolith/pkg/authz"
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
	"github.com/govalues/decimal"
)

type Mode uint8

const (
	Browsing Mode = iota
	Viewing
	Placing
	Tagging
)

type Filter struct {
	Status     models.OrderStatus
	Expression string
	Limit      int
}

type MenuOption struct {
	ID   entity.MenuID
	Name string
}
type DrinkOption struct {
	ID           entity.DrinkID
	Name         string
	Availability menumodels.Availability
	Price        string
}
type PlaceItem struct {
	DrinkID  entity.DrinkID
	Name     string
	Quantity int
	Notes    string
}
type Form struct {
	MenuQuery, DrinkQuery, Notes, Tags string
	MenuID                             entity.MenuID
	Items                              []PlaceItem
	ReplaceTags                        bool
}
type Line struct {
	DrinkID      entity.DrinkID
	Name         string
	Quantity     int
	Notes, Total string
}
type Row struct {
	Order    models.Order
	MenuName string
	Lines    []Line
	Total    string
}
type State struct {
	Mode                                            Mode
	Loading, CatalogLoading, Submitting, Confirming bool
	Rows                                            []Row
	Selected                                        *Row
	Filter                                          Filter
	Cursor, Next                                    paging.Cursor
	History                                         []paging.Cursor
	Form                                            Form
	Menus                                           []MenuOption
	Drinks                                          []DrinkOption
	Err                                             error
	CanPlace, CanComplete, CanCancel, CanTag        bool
	Dirty                                           bool
}
type Dependencies struct {
	Executor   ui.Executor
	Dispatcher ui.Dispatcher
	Dialogs    ui.Dialogs
}
type listResult struct {
	rows []Row
	next paging.Cursor
}
type placeCatalog struct {
	menus  []MenuOption
	drinks map[entity.MenuID][]DrinkOption
}

type Presenter struct {
	app           *app.Session
	dialogs       ui.Dialogs
	load          *ui.LatestRequest[listResult]
	catalog       *ui.LatestRequest[placeCatalog]
	submit        *ui.Submission
	state         State
	menuDrinks    map[entity.MenuID][]DrinkOption
	changed       func(State)
	confirmTarget *models.Order
}

func NewPresenter(session *app.Session, deps Dependencies) *Presenter {
	p := &Presenter{app: session, dialogs: deps.Dialogs, menuDrinks: make(map[entity.MenuID][]DrinkOption)}
	p.state.Filter.Limit = ui.PageLimit
	p.state.CanPlace = pkgAuthz.AuthorizeWithEntity(session.Context().Principal(), ordersauthz.ActionPlace, (models.Order{}).CedarEntity()) == nil
	p.load = ui.NewLatestRequest[listResult](deps.Executor, deps.Dispatcher)
	p.catalog = ui.NewLatestRequest[placeCatalog](deps.Executor, deps.Dispatcher)
	p.submit = ui.NewSubmission(deps.Executor, deps.Dispatcher)
	return p
}
func (p *Presenter) Observe(fn func(State)) { p.changed = fn; p.publish() }
func (p *Presenter) State() State           { return cloneState(p.state) }

func (p *Presenter) Refresh() {
	f, cursor := p.state.Filter, p.state.Cursor
	p.load.LoadContext(p.app.Context(), func(ctx context.Context) (listResult, error) {
		op := p.app.ContextFrom(ctx)
		page, err := p.app.Orders.List(op, orders.ListRequest{Status: f.Status, Filter: strings.TrimSpace(f.Expression), Cursor: cursor, Limit: f.Limit})
		if err != nil {
			return listResult{}, err
		}
		rows := make([]Row, 0, len(page.Items))
		for _, order := range page.Items {
			if order == nil {
				return listResult{}, apperrors.Internalf("order missing")
			}
			row, err := p.resolve(op, *order)
			if err != nil {
				return listResult{}, err
			}
			rows = append(rows, row)
		}
		return listResult{rows: rows, next: page.Next}, nil
	}, func(r ui.LoadState[listResult]) {
		p.state.Loading = r.Status == ui.Loading
		if r.Status == ui.Failed {
			p.state.Err = ui.PresentError(r.Err)
			ui.ShowPresentation(p.dialogs, r.Err)
		}
		if r.Status == ui.Loaded {
			selected := selectedID(p.state.Selected)
			if cursor == "" {
				p.state.Rows = cloneRows(r.Value.rows)
			} else {
				p.state.Rows = append(p.state.Rows, cloneRows(r.Value.rows)...)
			}
			p.state.Next, p.state.Err = r.Value.next, nil
			p.state.Selected = findRow(p.state.Rows, selected)
			p.permissions()
		}
		p.publish()
	})
}
func (p *Presenter) ApplyFilter(filter Filter) bool {
	if p.busy() {
		return false
	}
	if filter.Limit <= 0 {
		p.fail(apperrors.Invalidf("page size must be > 0"))
		return false
	}
	filter.Expression = strings.TrimSpace(filter.Expression)
	p.state.Filter, p.state.Cursor, p.state.Next, p.state.History = filter, "", "", nil
	p.Refresh()
	return true
}
func (p *Presenter) NextPage() {
	if p.busy() || p.state.Next == "" {
		return
	}
	p.state.History = append(p.state.History, p.state.Cursor)
	p.state.Cursor = p.state.Next
	p.Refresh()
}
func (p *Presenter) PreviousPage() {
	if p.busy() || len(p.state.History) == 0 {
		return
	}
	n := len(p.state.History) - 1
	p.state.Cursor = p.state.History[n]
	p.state.History = p.state.History[:n]
	p.Refresh()
}
func (p *Presenter) Select(index int) {
	if p.busy() || p.state.Mode != Browsing {
		return
	}
	if index < 0 || index >= len(p.state.Rows) {
		p.state.Selected = nil
	} else {
		row := cloneRow(p.state.Rows[index])
		p.state.Selected = &row
		p.state.Mode = Viewing
	}
	p.permissions()
	p.publish()
}

func (p *Presenter) ListPermissions(index int) (complete, cancel, tags bool) {
	if index < 0 || index >= len(p.state.Rows) {
		return false, false, false
	}
	order := p.state.Rows[index].Order
	principal, resource := p.app.Context().Principal(), order.CedarEntity()
	pending := order.Status == models.OrderStatusPending
	return pending && pkgAuthz.AuthorizeWithEntity(principal, ordersauthz.ActionComplete, resource) == nil,
		pending && pkgAuthz.AuthorizeWithEntity(principal, ordersauthz.ActionCancel, resource) == nil,
		pkgAuthz.AuthorizeWithEntity(principal, ordersauthz.ActionTag, resource) == nil
}

// Back returns to the exact list state from which the order was opened.
func (p *Presenter) Back() { p.leaveDetail(false) }

// ResetList is used by the breadcrumb and main navigation. It deliberately
// clears filter and paging state rather than preserving the prior search.
func (p *Presenter) ResetList() { p.leaveDetail(true) }

func (p *Presenter) leaveDetail(reset bool) {
	if p.state.Submitting || p.state.Confirming {
		return
	}
	proceed := func() {
		p.catalog.Invalidate()
		p.state.Mode, p.state.Form, p.state.Menus, p.state.Drinks = Browsing, Form{}, nil, nil
		p.state.Err, p.state.Dirty, p.state.Confirming = nil, false, false
		if reset {
			p.state.Selected = nil
			p.state.Filter, p.state.Cursor, p.state.Next, p.state.History = Filter{Limit: ui.PageLimit}, "", "", nil
		}
		p.permissions()
		p.publish()
		if reset {
			p.Refresh()
		}
	}
	if !p.state.Dirty {
		proceed()
		return
	}
	if p.dialogs == nil {
		return
	}
	p.state.Confirming = true
	p.publish()
	p.dialogs.Confirm("Discard changes?", "Discard unsaved order changes?", func(ok bool) {
		p.state.Confirming = false
		if ok {
			proceed()
			return
		}
		p.publish()
	})
}

func (p *Presenter) StartPlace() {
	if p.busy() || !p.state.CanPlace {
		return
	}
	p.state.Mode, p.state.Form, p.state.Menus, p.state.Drinks, p.state.Err = Placing, Form{ReplaceTags: true}, nil, nil, nil
	p.state.Dirty = false
	p.publish()
	p.loadPlaceCatalog()
}
func (p *Presenter) loadPlaceCatalog() {
	p.catalog.LoadContext(p.app.Context(), func(ctx context.Context) (placeCatalog, error) {
		op := p.app.ContextFrom(ctx)
		all, err := paging.Collect(func(cursor paging.Cursor) (paging.Page[*menumodels.Menu], error) {
			return p.app.Menus.List(op, menus.ListRequest{Status: menumodels.MenuStatusPublished, Cursor: cursor})
		})
		if err != nil {
			return placeCatalog{}, err
		}
		result := placeCatalog{drinks: make(map[entity.MenuID][]DrinkOption)}
		for _, menu := range all {
			if menu == nil {
				return placeCatalog{}, apperrors.Internalf("menu missing")
			}
			result.menus = append(result.menus, MenuOption{ID: menu.ID, Name: menu.Name})
			for _, item := range menu.Items {
				if item.Availability == menumodels.AvailabilityUnavailable {
					continue
				}
				name, ok := item.DisplayName.Unwrap()
				if !ok || strings.TrimSpace(name) == "" {
					drink, getErr := p.app.Drinks.Get(op, item.DrinkID)
					if getErr != nil {
						return placeCatalog{}, getErr
					}
					if drink == nil {
						return placeCatalog{}, apperrors.Internalf("drink %s missing", item.DrinkID)
					}
					name = drink.Name
				}
				price := "N/A"
				if value, ok := item.Price.Unwrap(); ok {
					price = value.String()
				}
				result.drinks[menu.ID] = append(result.drinks[menu.ID], DrinkOption{ID: item.DrinkID, Name: name, Availability: item.Availability, Price: price})
			}
			sort.Slice(result.drinks[menu.ID], func(i, j int) bool {
				return optionLess(result.drinks[menu.ID][i].Name, result.drinks[menu.ID][i].ID.String(), result.drinks[menu.ID][j].Name, result.drinks[menu.ID][j].ID.String())
			})
		}
		sort.Slice(result.menus, func(i, j int) bool {
			return optionLess(result.menus[i].Name, result.menus[i].ID.String(), result.menus[j].Name, result.menus[j].ID.String())
		})
		return result, nil
	}, func(r ui.LoadState[placeCatalog]) {
		if p.state.Mode != Placing {
			return
		}
		p.state.CatalogLoading = r.Status == ui.Loading
		if r.Status == ui.Failed {
			p.state.Err = ui.PresentError(r.Err)
			ui.ShowPresentation(p.dialogs, r.Err)
		}
		if r.Status == ui.Loaded {
			p.menuDrinks = cloneDrinkMap(r.Value.drinks)
			p.state.Menus = filterMenus(r.Value.menus, p.state.Form.MenuQuery)
			p.state.Drinks = filterDrinks(p.menuDrinks[p.state.Form.MenuID], p.state.Form.DrinkQuery)
			p.state.Err = nil
		}
		p.publish()
	})
}
func (p *Presenter) SearchMenus(query string) {
	if p.state.Mode != Placing || p.state.Submitting {
		return
	}
	p.state.Form.MenuQuery = query
	// Reload against public APIs so newly published menus are searchable. The
	// form lives independently from the catalog result and remains dirty-safe.
	p.publish()
	p.loadPlaceCatalog()
}
func (p *Presenter) ChooseMenu(id entity.MenuID) {
	if p.state.Mode != Placing || p.state.Submitting {
		return
	}
	p.state.Form.MenuID, p.state.Form.Items = id, nil
	p.state.Dirty = true
	p.state.Drinks = filterDrinks(p.menuDrinks[id], p.state.Form.DrinkQuery)
	p.state.Err = nil
	p.publish()
}
func (p *Presenter) SearchDrinks(query string) {
	if p.state.Mode != Placing || p.state.Submitting {
		return
	}
	p.state.Form.DrinkQuery = query
	p.state.Drinks = filterDrinks(p.menuDrinks[p.state.Form.MenuID], query)
	p.publish()
}
func (p *Presenter) AddItem(id entity.DrinkID, quantity int, notes string) bool {
	if p.state.Mode != Placing || p.state.Submitting {
		return false
	}
	if p.state.Form.MenuID.IsZero() {
		p.fail(apperrors.Invalidf("menu is required"))
		return false
	}
	if quantity <= 0 {
		p.fail(apperrors.Invalidf("quantity must be > 0"))
		return false
	}
	var found *DrinkOption
	for i := range p.menuDrinks[p.state.Form.MenuID] {
		if p.menuDrinks[p.state.Form.MenuID][i].ID == id {
			v := p.menuDrinks[p.state.Form.MenuID][i]
			found = &v
			break
		}
	}
	if found == nil {
		p.fail(apperrors.Invalidf("available menu drink is required"))
		return false
	}
	for i := range p.state.Form.Items {
		if p.state.Form.Items[i].DrinkID == id {
			p.state.Form.Items[i].Quantity += quantity
			p.state.Form.Items[i].Notes = strings.TrimSpace(notes)
			p.state.Dirty = true
			p.state.Err = nil
			p.publish()
			return true
		}
	}
	p.state.Form.Items = append(p.state.Form.Items, PlaceItem{DrinkID: id, Name: found.Name, Quantity: quantity, Notes: strings.TrimSpace(notes)})
	p.state.Dirty = true
	p.state.Err = nil
	p.publish()
	return true
}
func (p *Presenter) RemoveItem(index int) {
	if p.state.Mode != Placing || p.state.Submitting || index < 0 || index >= len(p.state.Form.Items) {
		return
	}
	p.state.Form.Items = append(p.state.Form.Items[:index:index], p.state.Form.Items[index+1:]...)
	p.state.Dirty = true
	p.publish()
}
func (p *Presenter) SetPlaceNotes(notes string) {
	if p.state.Mode == Placing && !p.state.Submitting {
		if p.state.Form.Notes == notes {
			return
		}
		p.state.Form.Notes = notes
		p.state.Dirty = true
		p.publish()
	}
}
func (p *Presenter) SetPlaceTags(tags string) {
	if p.state.Mode == Placing && !p.state.Submitting {
		if p.state.Form.Tags == tags && p.state.Form.ReplaceTags {
			return
		}
		p.state.Form.Tags, p.state.Form.ReplaceTags = tags, true
		p.state.Dirty = true
		p.publish()
	}
}
func (p *Presenter) SavePlace() bool {
	if p.state.Mode != Placing || p.state.Submitting {
		return false
	}
	form := cloneForm(p.state.Form)
	if form.MenuID.IsZero() {
		p.fail(apperrors.Invalidf("menu is required"))
		return false
	}
	if len(form.Items) == 0 {
		p.fail(apperrors.Invalidf("order must have at least 1 item"))
		return false
	}
	items := make([]models.OrderItem, len(form.Items))
	for i, item := range form.Items {
		if item.Quantity <= 0 {
			p.fail(apperrors.Invalidf("item %d: quantity must be > 0", i))
			return false
		}
		items[i] = models.OrderItem{DrinkID: item.DrinkID, Quantity: item.Quantity, Notes: strings.TrimSpace(item.Notes)}
	}
	desired, err := orderTagChoice(ui.ReplaceTags, form.Tags)
	if !form.ReplaceTags {
		desired, err = nil, nil
	}
	if err != nil {
		p.fail(err)
		return false
	}
	return p.mutate(func() error {
		_, err := app.RunTaggedMutation(p.app.App, p.app.Context(), desired, func(ctx *middleware.Context) (*models.Order, error) {
			return p.app.Orders.Place(ctx, &models.Order{MenuID: form.MenuID, Items: items, Notes: strings.TrimSpace(form.Notes)})
		})
		return err
	}, true)
}
func (p *Presenter) StartTags() {
	if p.busy() || p.state.Selected == nil || !p.state.CanTag {
		return
	}
	text, _ := tag.FormatCollection(p.state.Selected.Order.Tags)
	p.state.Mode, p.state.Form, p.state.Err = Tagging, Form{Tags: text}, nil
	p.state.Dirty = false
	p.publish()
}
func (p *Presenter) SetTagForm(value string) {
	if p.state.Mode != Tagging || p.state.Submitting || p.state.Selected == nil {
		return
	}
	p.state.Form.Tags = value
	current, _ := tag.FormatCollection(p.state.Selected.Order.Tags)
	p.state.Dirty = value != current
	p.publish()
}
func (p *Presenter) SaveTags(value string) bool {
	if p.state.Mode != Tagging || p.state.Selected == nil || p.state.Submitting {
		return false
	}
	// Preserve exactly what the user entered before validation. A retained-mode
	// render follows fail(), so updating the form first keeps an invalid value
	// available for correction instead of silently restoring the old tags.
	p.state.Form.Tags = value
	tags, err := tag.ParseCollection(value)
	if err != nil {
		p.fail(err)
		return false
	}
	target := p.state.Selected.Order
	return p.mutate(func() error { _, err := p.app.Tags.Replace(p.app.Context(), target.EntityUID(), tags); return err }, true)
}
func (p *Presenter) CancelForm() {
	if p.state.Submitting {
		return
	}
	p.catalog.Invalidate()
	mode := Browsing
	if p.state.Selected != nil {
		mode = Viewing
	}
	p.state.Mode, p.state.Form, p.state.Menus, p.state.Drinks, p.state.Err, p.state.Dirty = mode, Form{}, nil, nil, nil, false
	p.publish()
}

func (p *Presenter) ConfirmComplete() {
	if p.state.CanComplete {
		p.confirm("Complete order", models.OrderStatusCompleted)
	}
}
func (p *Presenter) ConfirmCancel() {
	if p.state.CanCancel {
		p.confirm("Cancel order", models.OrderStatusCancelled)
	}
}
func (p *Presenter) confirm(title string, status models.OrderStatus) {
	if p.busy() || p.state.Selected == nil || p.state.Selected.Order.Status != models.OrderStatusPending {
		return
	}
	target := p.state.Selected.Order
	p.confirmTarget = cloneOrder(&target)
	p.state.Confirming = true
	p.publish()
	ui.ConfirmTagged(p.dialogs, title, fmt.Sprintf("%s %s?", title, target.ID), target.Tags.Canonical().String(), func(ok bool, mode ui.TagMutationMode, values string) {
		stable := cloneOrder(p.confirmTarget)
		p.confirmTarget = nil
		p.state.Confirming = false
		if !ok {
			p.publish()
			return
		}
		desired, err := orderTagChoice(mode, values)
		if err != nil {
			p.fail(err)
			return
		}
		p.mutate(func() error {
			if status == models.OrderStatusCompleted {
				_, err := app.RunTaggedMutation(p.app.App, p.app.Context(), desired, func(ctx *middleware.Context) (*models.Order, error) { return p.app.Orders.Complete(ctx, stable) })
				return err
			}
			_, err := app.RunTaggedMutation(p.app.App, p.app.Context(), desired, func(ctx *middleware.Context) (*models.Order, error) { return p.app.Orders.Cancel(ctx, stable) })
			return err
		}, false)
	})
}

func orderTagChoice(mode ui.TagMutationMode, values string) (*tag.Tags, error) {
	if mode == ui.PreserveTags {
		return nil, nil
	}
	if mode == ui.ClearTags {
		empty := tag.Tags{}
		return &empty, nil
	}
	parsed, err := tag.ParseCollection(values)
	return &parsed, err
}
func (p *Presenter) mutate(work func() error, closeForm bool) bool {
	p.state.Submitting = true
	p.state.Err = nil
	p.publish()
	accepted := p.submit.Submit(work, func(err error) {
		p.state.Submitting = false
		p.state.Err = ui.PresentError(err)
		if err == nil && closeForm {
			if p.state.Selected != nil {
				p.state.Mode = Viewing
			} else {
				p.state.Mode = Browsing
			}
			p.state.Form = Form{}
			p.state.Dirty = false
		}
		p.publish()
		ui.ShowPresentation(p.dialogs, err)
		if err == nil {
			p.Refresh()
		}
	})
	if !accepted {
		p.state.Submitting = p.submit.Active()
		p.publish()
	}
	return accepted
}

func (p *Presenter) permissions() {
	p.state.CanComplete, p.state.CanCancel, p.state.CanTag = false, false, false
	if p.state.Selected == nil {
		return
	}
	principal, resource := p.app.Context().Principal(), p.state.Selected.Order.CedarEntity()
	p.state.CanComplete = pkgAuthz.AuthorizeWithEntity(principal, ordersauthz.ActionComplete, resource) == nil
	p.state.CanCancel = pkgAuthz.AuthorizeWithEntity(principal, ordersauthz.ActionCancel, resource) == nil
	p.state.CanTag = pkgAuthz.AuthorizeWithEntity(principal, ordersauthz.ActionTag, resource) == nil
}

func (p *Presenter) resolve(ctx *middleware.Context, order models.Order) (Row, error) {
	menu, err := p.app.Menus.Get(ctx, order.MenuID)
	if err != nil {
		return Row{}, err
	}
	if menu == nil {
		return Row{}, apperrors.Internalf("menu %s missing", order.MenuID)
	}
	menuItems := make(map[entity.DrinkID]menumodels.MenuItem, len(menu.Items))
	for _, item := range menu.Items {
		menuItems[item.DrinkID] = item
	}
	items := append([]models.OrderItem(nil), order.Items...)
	sort.Slice(items, func(i, j int) bool { return items[i].DrinkID.String() < items[j].DrinkID.String() })
	row := Row{Order: *cloneOrder(&order), MenuName: menu.Name, Total: "N/A"}
	var total menumodels.Price
	totalSet, available := false, true
	for _, item := range items {
		mi, ok := menuItems[item.DrinkID]
		name := item.DrinkID.String()
		if displayName, hasDisplayName := mi.DisplayName.Unwrap(); ok && hasDisplayName && strings.TrimSpace(displayName) != "" {
			name = displayName
		} else {
			drink, getErr := p.app.Drinks.Get(ctx, item.DrinkID)
			if getErr != nil && !apperrors.IsPermission(getErr) {
				return Row{}, getErr
			}
			if getErr == nil {
				if drink == nil {
					return Row{}, apperrors.Internalf("drink %s missing", item.DrinkID)
				}
				name = drink.Name
			}
		}
		line := Line{DrinkID: item.DrinkID, Name: name, Quantity: item.Quantity, Notes: item.Notes, Total: "N/A"}
		price, hasPrice := mi.Price.Unwrap()
		if !ok || !hasPrice {
			available = false
		} else {
			qty, _ := decimal.New(int64(item.Quantity), 0)
			amount, mulErr := price.Mul(qty)
			if mulErr != nil {
				return Row{}, mulErr
			}
			line.Total = amount.String()
			if !totalSet {
				total = amount
				totalSet = true
			} else {
				total, err = total.Add(amount)
				if err != nil {
					return Row{}, err
				}
			}
		}
		row.Lines = append(row.Lines, line)
	}
	if available && totalSet {
		row.Total = total.String()
	}
	return row, nil
}
func (p *Presenter) busy() bool {
	return p.state.Loading || p.state.CatalogLoading || p.state.Submitting || p.state.Confirming
}
func (p *Presenter) fail(err error) {
	p.state.Err = ui.PresentError(err)
	ui.ShowPresentation(p.dialogs, err)
	p.publish()
}
func (p *Presenter) publish() {
	if p.changed != nil {
		p.changed(cloneState(p.state))
	}
}

func optionLess(an, ai, bn, bi string) bool {
	a, b := strings.ToLower(an), strings.ToLower(bn)
	if a == b {
		return ai < bi
	}
	return a < b
}
func filterMenus(in []MenuOption, q string) []MenuOption {
	q = strings.ToLower(strings.TrimSpace(q))
	out := make([]MenuOption, 0, len(in))
	for _, v := range in {
		if q == "" || strings.Contains(strings.ToLower(v.Name), q) {
			out = append(out, v)
		}
	}
	return out
}
func filterDrinks(in []DrinkOption, q string) []DrinkOption {
	q = strings.ToLower(strings.TrimSpace(q))
	out := make([]DrinkOption, 0, len(in))
	for _, v := range in {
		if q == "" || strings.Contains(strings.ToLower(v.Name), q) {
			out = append(out, v)
		}
	}
	return out
}
func selectedID(row *Row) entity.OrderID {
	if row == nil {
		return entity.OrderID{}
	}
	return row.Order.ID
}
func findRow(rows []Row, id entity.OrderID) *Row {
	for _, r := range rows {
		if r.Order.ID == id {
			x := cloneRow(r)
			return &x
		}
	}
	return nil
}
func cloneOrder(in *models.Order) *models.Order {
	if in == nil {
		return nil
	}
	out := *in
	out.Items = append([]models.OrderItem(nil), in.Items...)
	out.Tags = append(tag.Tags(nil), in.Tags...)
	return &out
}
func cloneRow(in Row) Row {
	out := in
	out.Order = *cloneOrder(&in.Order)
	out.Lines = append([]Line(nil), in.Lines...)
	return out
}
func cloneRows(in []Row) []Row {
	out := make([]Row, len(in))
	for i := range in {
		out[i] = cloneRow(in[i])
	}
	return out
}
func cloneForm(in Form) Form {
	out := in
	out.Items = append([]PlaceItem(nil), in.Items...)
	return out
}
func cloneState(in State) State {
	out := in
	out.Rows = cloneRows(in.Rows)
	out.Selected = nil
	if in.Selected != nil {
		x := cloneRow(*in.Selected)
		out.Selected = &x
	}
	out.History = append([]paging.Cursor(nil), in.History...)
	out.Form = cloneForm(in.Form)
	out.Menus = append([]MenuOption(nil), in.Menus...)
	out.Drinks = append([]DrinkOption(nil), in.Drinks...)
	return out
}
func cloneDrinkMap(in map[entity.MenuID][]DrinkOption) map[entity.MenuID][]DrinkOption {
	out := make(map[entity.MenuID][]DrinkOption, len(in))
	for k, v := range in {
		out[k] = append([]DrinkOption(nil), v...)
	}
	return out
}
func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}
