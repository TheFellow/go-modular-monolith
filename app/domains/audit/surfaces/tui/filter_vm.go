package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	tuikeys "github.com/TheFellow/go-modular-monolith/main/tui/keys"
	tuistyles "github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type auditScope string

const (
	scopeAll    auditScope = "all activity"
	scopeEntity auditScope = "entity history"
	scopeActor  auditScope = "actor activity"
)

type auditQuery struct {
	scope                                           auditScope
	entity, principal, action, from, to, expression string
	limit                                           int
	cursor                                          paging.Cursor
}

type filterVM struct {
	form                                            *forms.Form
	scope                                           *forms.SelectField
	entity, principal, action, from, to, expression *forms.TextField
	limit                                           *forms.NumberField
	err                                             error
	viewport                                        viewport.Model
	height                                          int
}

func newFilterVM(q auditQuery) *filterVM {
	if q.scope == "" {
		q.scope = scopeAll
	}
	if q.limit <= 0 {
		q.limit = paging.DefaultLimit
	}
	v := &filterVM{
		scope:      forms.NewSelectField("Scope", []forms.SelectOption{{Label: string(scopeAll), Value: scopeAll}, {Label: string(scopeEntity), Value: scopeEntity}, {Label: string(scopeActor), Value: scopeActor}}, forms.WithInitialValue(q.scope)),
		entity:     forms.NewTextField("Entity (history)", forms.WithInitialValue(q.entity)),
		principal:  forms.NewTextField("Principal (actor/all)", forms.WithInitialValue(q.principal)),
		action:     forms.NewTextField("Action (all only)", forms.WithInitialValue(q.action)),
		from:       forms.NewTextField("From (RFC3339 or YYYY-MM-DD)", forms.WithInitialValue(q.from)),
		to:         forms.NewTextField("To (RFC3339 or YYYY-MM-DD)", forms.WithInitialValue(q.to)),
		expression: forms.NewTextField("Expression", forms.WithInitialValue(q.expression)),
		limit:      forms.NewNumberField("Page size", forms.WithRequired(), forms.WithMin(1), forms.WithInitialValue(q.limit)),
	}
	v.form = forms.New(tuistyles.App.Form, tuikeys.App.Form, v.scope, v.entity, v.principal, v.action, v.from, v.to, v.expression, v.limit)
	v.viewport = viewport.New(0, 0)
	v.configureForm(q.scope)
	return v
}
func (v *filterVM) Init() tea.Cmd { return v.form.Init() }
func (v *filterVM) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	v.form, cmd = v.form.Update(msg)
	if scope, ok := v.scope.Value().(auditScope); ok && !v.scopeInForm(scope) {
		v.configureForm(scope)
		return v.form.Init()
	}
	return cmd
}

func (v *filterVM) configureForm(scope auditScope) {
	fields := []forms.Field{v.scope}
	switch scope { //nolint:exhaustive // scopeAll deliberately contributes no structured predicate.
	case scopeEntity:
		fields = append(fields, v.entity)
	case scopeActor:
		fields = append(fields, v.principal)
	default:
		fields = append(fields, v.entity, v.principal, v.action)
	}
	fields = append(fields, v.from, v.to, v.expression, v.limit)
	v.form = forms.New(tuistyles.App.Form, tuikeys.App.Form, fields...)
	if v.viewport.Width > 0 {
		v.form.SetWidth(max(v.viewport.Width-2, 1))
	}
}
func (v *filterVM) scopeInForm(scope auditScope) bool {
	want := 7
	if scope == scopeEntity || scope == scopeActor {
		want = 5
	}
	// The focused field is not enough to identify shape; the rendered form is
	// the surface contract and these labels are unique to each scope.
	view := v.form.View()
	if want == 7 {
		return strings.Contains(view, "Action (all only)")
	}
	if scope == scopeEntity {
		return strings.Contains(view, "Entity (history)") && !strings.Contains(view, "Principal (actor/all)")
	}
	return strings.Contains(view, "Principal (actor/all)") && !strings.Contains(view, "Entity (history)")
}
func (v *filterVM) View() string {
	out := "Query Audit\n\n" + v.form.View() + "\n\nScope rules: history uses entity; actor uses principal; all uses entity/principal/action."
	if v.err != nil {
		out = "Error: " + v.err.Error() + "\n\n" + out
	}
	if v.height <= 0 {
		return out + "\nctrl+s apply • esc cancel"
	}
	v.viewport.SetContent(out)
	if field := v.form.FocusedField(); field != nil {
		if at := strings.Index(out, field.Label()); at >= 0 {
			line := strings.Count(out[:at], "\n")
			if line < v.viewport.YOffset {
				v.viewport.SetYOffset(line)
			}
			if line+2 >= v.viewport.YOffset+v.viewport.Height {
				v.viewport.SetYOffset(max(line-v.viewport.Height+3, 0))
			}
		}
	}
	return v.viewport.View() + "\nctrl+s apply • esc cancel"
}
func (v *filterVM) SetSize(width, height int) {
	v.form.SetWidth(max(width-2, 1))
	v.height = height
	v.viewport.Width = max(width, 1)
	v.viewport.Height = max(height-1, 1)
}
func (v *filterVM) Query() (auditQuery, audit.ListRequest, error) {
	if err := v.form.Validate(); err != nil {
		v.err = err
		return auditQuery{}, audit.ListRequest{}, err
	}
	limit, err := strconv.Atoi(fmt.Sprint(v.limit.Value()))
	if err != nil || limit <= 0 {
		v.err = apperrors.Invalidf("page size must be greater than zero")
		return auditQuery{}, audit.ListRequest{}, v.err
	}
	q := auditQuery{scope: v.scope.Value().(auditScope), entity: strings.TrimSpace(fmt.Sprint(v.entity.Value())), principal: strings.TrimSpace(fmt.Sprint(v.principal.Value())), action: strings.TrimSpace(fmt.Sprint(v.action.Value())), from: strings.TrimSpace(fmt.Sprint(v.from.Value())), to: strings.TrimSpace(fmt.Sprint(v.to.Value())), expression: strings.TrimSpace(fmt.Sprint(v.expression.Value())), limit: limit}
	req, err := q.Request()
	if err != nil {
		v.err = err
		return auditQuery{}, audit.ListRequest{}, err
	}
	v.err = nil
	return q, req, nil
}
func (q auditQuery) Request() (audit.ListRequest, error) {
	req := audit.ListRequest{Cursor: q.cursor, Limit: q.limit, Filter: q.expression}
	var err error
	switch q.scope {
	case scopeAll:
		req.Entity, err = parseAuditUID(q.entity)
		if err != nil {
			return req, err
		}
		req.Principal, err = parseAuditPrincipal(q.principal)
		if err != nil {
			return req, err
		}
		req.Action, err = parseAuditUID(q.action)
	case scopeEntity:
		if q.entity == "" {
			return req, apperrors.Invalidf("entity is required for history")
		}
		req.Entity, err = parseAuditUID(q.entity)
	case scopeActor:
		if q.principal == "" {
			return req, apperrors.Invalidf("principal is required for actor activity")
		}
		req.Principal, err = parseAuditPrincipal(q.principal)
	default:
		return req, apperrors.Invalidf("invalid audit scope")
	}
	if err != nil {
		return req, err
	}
	req.From, err = parseAuditTime(q.from)
	if err != nil {
		return req, err
	}
	req.To, err = parseAuditTime(q.to)
	if err != nil {
		return req, err
	}
	return req, nil
}
func parseAuditTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	if t, e := time.Parse(time.RFC3339, value); e == nil {
		return t, nil
	}
	if t, e := time.Parse("2006-01-02", value); e == nil {
		return t, nil
	}
	return time.Time{}, apperrors.Invalidf("invalid time %q", value)
}
func parseAuditPrincipal(value string) (cedar.EntityUID, error) {
	if value == "" {
		return cedar.EntityUID{}, nil
	}
	if !strings.Contains(value, "::") {
		if p, e := authn.ParseActor(value); e == nil {
			return p, nil
		}
	}
	return parseAuditUID(value)
}
func parseAuditUID(value string) (cedar.EntityUID, error) {
	if value == "" {
		return cedar.EntityUID{}, nil
	}
	if strings.Contains(value, "::\"") || strings.HasSuffix(value, "\"") {
		var uid cedar.EntityUID
		if err := uid.UnmarshalCedar([]byte(value)); err != nil {
			return uid, apperrors.Invalidf("invalid entity uid %q: %v", value, err)
		}
		return uid, nil
	}
	at := strings.LastIndex(value, "::")
	if at <= 0 || at+2 >= len(value) {
		return cedar.EntityUID{}, apperrors.Invalidf("invalid entity uid %q", value)
	}
	typ, id := value[:at], strings.Trim(value[at+2:], "\"")
	if typ == "" || id == "" {
		return cedar.EntityUID{}, apperrors.Invalidf("invalid entity uid %q", value)
	}
	return cedar.NewEntityUID(cedar.EntityType(typ), cedar.String(id)), nil
}
