// Package gui provides the bespoke retained-mode desktop presentation for audit.
package gui

import (
	"context"
	"maps"
	"sort"
	"strings"
	"time"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
	cedar "github.com/cedar-policy/cedar-go"
)

const dateLayout = "2006-01-02"

type Scope uint8

const (
	AllActivity Scope = iota
	EntityHistory
	ActorActivity
)

type Mode uint8

const (
	Browsing Mode = iota
	Viewing
)

func (s Scope) valid() bool {
	switch s {
	case AllActivity, EntityHistory, ActorActivity:
		return true
	}
	return false
}

type Filter struct {
	Scope                               Scope
	Entity, Principal, Action, From, To string
	Expression                          string
	Limit                               int
}

type Row struct {
	Entry   models.AuditEntry
	Touches []string
	Actions map[actions.ID]actions.State
}

type State struct {
	Mode         Mode
	Loading      bool
	Rows         []Row
	Selected     *Row
	Filter       Filter
	Cursor, Next paging.Cursor
	History      []paging.Cursor
	Err          error
	Actions      map[actions.ID]actions.State
}

type Dependencies struct {
	Executor   ui.Executor
	Dispatcher ui.Dispatcher
	Dialogs    ui.Dialogs
	Projector  *audit.ActionProjector
}

type listResult struct {
	rows []Row
	next paging.Cursor
}

type Presenter struct {
	app       *app.Session
	dialogs   ui.Dialogs
	load      *ui.LatestRequest[listResult]
	state     State
	changed   func(State)
	projector audit.ActionProjector
}

func NewPresenter(session *app.Session, deps Dependencies) *Presenter {
	projector := audit.NewActionProjector()
	if deps.Projector != nil {
		projector = *deps.Projector
	}
	p := &Presenter{app: session, dialogs: deps.Dialogs, projector: projector}
	p.state.Filter.Limit = ui.PageLimit
	p.load = ui.NewLatestRequest[listResult](deps.Executor, deps.Dispatcher)
	states, err := projector.Project(session.Context(), session.Context().Principal(), nil)
	if err != nil {
		p.state.Err = ui.PresentError(err)
	} else {
		p.state.Actions = indexActions(states)
	}
	return p
}

func (p *Presenter) Observe(fn func(State)) { p.changed = fn; p.publish() }
func (p *Presenter) State() State           { return cloneState(p.state) }

func (p *Presenter) Refresh() {
	if !p.actionEnabled(audit.ControlList) {
		return
	}
	p.state.Cursor, p.state.Next, p.state.History = "", "", nil
	p.loadPage(false)
}

func (p *Presenter) loadPage(appendPage bool) {
	filter, cursor := p.state.Filter, p.state.Cursor
	req, err := requestFromFilter(filter, cursor)
	if err != nil {
		p.fail(err)
		return
	}
	p.load.LoadContext(p.app.Context(), func(ctx context.Context) (listResult, error) {
		page, err := p.app.Audit.List(p.app.ContextFrom(ctx), req)
		if err != nil {
			return listResult{}, err
		}
		rows := make([]Row, 0, len(page.Items))
		for _, entry := range page.Items {
			if entry == nil {
				return listResult{}, errors.Internalf("audit entry missing")
			}
			row := rowFromEntry(*entry)
			states, err := p.projector.Project(ctx, p.app.Context().Principal(), entry)
			if err != nil {
				return listResult{}, err
			}
			row.Actions = indexActions(states)
			rows = append(rows, row)
		}
		return listResult{rows: rows, next: page.Next}, nil
	}, func(result ui.LoadState[listResult]) {
		p.state.Loading = result.Status == ui.Loading
		if result.Status == ui.Failed {
			p.state.Err = ui.PresentError(result.Err)
			ui.ShowPresentation(p.dialogs, result.Err)
		}
		if result.Status == ui.Loaded {
			if appendPage {
				p.state.Rows = append(p.state.Rows, cloneRows(result.Value.rows)...)
			} else {
				p.state.Rows = cloneRows(result.Value.rows)
			}
			p.state.Next, p.state.Err = result.Value.next, nil
			if p.state.Mode == Viewing {
				p.state.Selected = findRow(p.state.Rows, selectedID(p.state.Selected))
			}
		}
		p.publish()
	})
}

func (p *Presenter) ApplyFilter(filter Filter) bool {
	if p.state.Loading || !p.actionEnabled(audit.ControlList) {
		return false
	}
	filter = normalizeFilter(filter)
	if _, err := requestFromFilter(filter, ""); err != nil {
		p.fail(err)
		return false
	}
	p.state.Filter, p.state.Cursor, p.state.Next, p.state.History = filter, "", "", nil
	p.loadPage(false)
	return true
}

func (p *Presenter) NextPage() {
	if p.state.Loading || p.state.Next == "" || !p.actionEnabled(audit.ControlList) {
		return
	}
	p.state.History = append(p.state.History, p.state.Cursor)
	p.state.Cursor = p.state.Next
	p.loadPage(true)
}

func (p *Presenter) PreviousPage() {
	if p.state.Loading || len(p.state.History) == 0 || !p.actionEnabled(audit.ControlList) {
		return
	}
	last := len(p.state.History) - 1
	p.state.Cursor = p.state.History[last]
	p.state.History = p.state.History[:last]
	p.loadPage(false)
}

func (p *Presenter) actionEnabled(id actions.ID) bool {
	state, ok := p.state.Actions[id]
	return ok && state.Visible && state.Enabled
}

func (p *Presenter) Select(index int) {
	if p.state.Loading {
		return
	}
	if index < 0 || index >= len(p.state.Rows) {
		p.state.Selected = nil
	} else if state, ok := p.state.Rows[index].Actions[audit.ControlView]; !ok || !state.Visible || !state.Enabled {
		return
	} else {
		row := cloneRow(p.state.Rows[index])
		p.state.Selected = &row
		p.state.Mode = Viewing
	}
	p.publish()
}

func indexActions(states []actions.State) map[actions.ID]actions.State {
	indexed := make(map[actions.ID]actions.State, len(states))
	for _, state := range states {
		indexed[state.ID] = state
	}
	return indexed
}

// Back returns to the exact list query and page that opened the detail.
func (p *Presenter) Back() {
	p.state.Mode, p.state.Selected = Browsing, nil
	p.publish()
}

// ResetList is used by the breadcrumb and main navigation: it deliberately
// discards the retained filter and paging state.
func (p *Presenter) ResetList() {
	p.state.Mode, p.state.Selected = Browsing, nil
	p.state.Filter = Filter{Limit: ui.PageLimit}
	p.state.Cursor, p.state.Next, p.state.History = "", "", nil
	p.Refresh()
}

func normalizeFilter(filter Filter) Filter {
	filter.Entity = strings.TrimSpace(filter.Entity)
	filter.Principal = strings.TrimSpace(filter.Principal)
	filter.Action = strings.TrimSpace(filter.Action)
	filter.From = strings.TrimSpace(filter.From)
	filter.To = strings.TrimSpace(filter.To)
	filter.Expression = strings.TrimSpace(filter.Expression)
	return filter
}

func requestFromFilter(filter Filter, cursor paging.Cursor) (audit.ListRequest, error) {
	if filter.Limit <= 0 {
		return audit.ListRequest{}, errors.Invalidf("page size must be greater than zero")
	}
	req := audit.ListRequest{Cursor: cursor, Limit: filter.Limit, Filter: filter.Expression}
	if !filter.Scope.valid() {
		return req, errors.Invalidf("invalid audit scope")
	}
	switch filter.Scope {
	case AllActivity:
	case EntityHistory:
		if filter.Entity == "" {
			return req, errors.Invalidf("entity is required for history")
		}
		filter.Principal, filter.Action = "", ""
	case ActorActivity:
		if filter.Principal == "" {
			return req, errors.Invalidf("principal is required for actor activity")
		}
		filter.Entity, filter.Action = "", ""
	}
	var err error
	if req.Entity, err = parseEntityUID(filter.Entity); err != nil {
		return req, err
	}
	if req.Principal, err = parsePrincipal(filter.Principal); err != nil {
		return req, err
	}
	if req.Action, err = parseEntityUID(filter.Action); err != nil {
		return req, err
	}
	if req.From, err = parseTime(filter.From); err != nil {
		return req, err
	}
	if req.To, err = parseTime(filter.To); err != nil {
		return req, err
	}
	return req, nil
}

func parseTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	if parsed, err := time.Parse(dateLayout, value); err == nil {
		return parsed, nil
	}
	return time.Time{}, errors.Invalidf("invalid time %q", value)
}

func parsePrincipal(value string) (cedar.EntityUID, error) {
	if value == "" {
		return cedar.EntityUID{}, nil
	}
	if !strings.Contains(value, "::") {
		if principal, err := authn.ParseActor(value); err == nil {
			return principal, nil
		}
	}
	return parseEntityUID(value)
}

func parseEntityUID(value string) (cedar.EntityUID, error) {
	if value == "" {
		return cedar.EntityUID{}, nil
	}
	if strings.Contains(value, "::\"") || strings.HasSuffix(value, "\"") {
		var uid cedar.EntityUID
		if err := uid.UnmarshalCedar([]byte(value)); err != nil {
			return cedar.EntityUID{}, errors.Invalidf("invalid entity uid %q: %v", value, err)
		}
		return uid, nil
	}
	index := strings.LastIndex(value, "::")
	if index <= 0 || index+2 >= len(value) {
		return cedar.EntityUID{}, errors.Invalidf("invalid entity uid %q", value)
	}
	typ, id := value[:index], strings.Trim(value[index+2:], "\"")
	if typ == "" || id == "" {
		return cedar.EntityUID{}, errors.Invalidf("invalid entity uid %q", value)
	}
	return cedar.NewEntityUID(cedar.EntityType(typ), cedar.String(id)), nil
}

func rowFromEntry(entry models.AuditEntry) Row {
	row := Row{Entry: cloneEntry(entry), Touches: make([]string, 0, len(entry.Touches))}
	for _, touched := range entry.Touches {
		row.Touches = append(row.Touches, touched.String())
	}
	sort.Strings(row.Touches)
	return row
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

func cloneEntry(entry models.AuditEntry) models.AuditEntry {
	entry.Touches = append([]cedar.EntityUID(nil), entry.Touches...)
	return entry
}
func cloneRow(row Row) Row {
	row.Entry = cloneEntry(row.Entry)
	row.Touches = append([]string(nil), row.Touches...)
	row.Actions = cloneActions(row.Actions)
	return row
}
func cloneRows(rows []Row) []Row {
	out := make([]Row, len(rows))
	for index := range rows {
		out[index] = cloneRow(rows[index])
	}
	return out
}
func cloneState(state State) State {
	state.Rows = cloneRows(state.Rows)
	state.History = append([]paging.Cursor(nil), state.History...)
	state.Actions = cloneActions(state.Actions)
	if state.Selected != nil {
		row := cloneRow(*state.Selected)
		state.Selected = &row
	}
	return state
}
func cloneActions(in map[actions.ID]actions.State) map[actions.ID]actions.State {
	if in == nil {
		return nil
	}
	out := make(map[actions.ID]actions.State, len(in))
	maps.Copy(out, in)
	return out
}
func selectedID(row *Row) string {
	if row == nil {
		return ""
	}
	return row.Entry.ID.String()
}
func findRow(rows []Row, id string) *Row {
	if id == "" {
		return nil
	}
	for _, row := range rows {
		if row.Entry.ID.String() == id {
			found := cloneRow(row)
			return &found
		}
	}
	return nil
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}
func formatDuration(start, completed time.Time) string {
	if start.IsZero() || completed.IsZero() || completed.Before(start) {
		return ""
	}
	return completed.Sub(start).Round(time.Microsecond).String()
}
