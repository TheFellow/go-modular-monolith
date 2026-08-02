package tui

import (
	"fmt"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"

	"github.com/TheFellow/go-modular-monolith/app"
	auditmodels "github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/keys"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type ListViewModel struct {
	app           *app.Session
	keys          keys.ListViewKeys
	formKeys      forms.FormKeys
	shell         *tui.ListDetail
	detail        *DetailViewModel
	query         auditQuery
	next          paging.Cursor
	history       []paging.Cursor
	filter        *filterVM
	loadToken     uint64
	width, height int
}

func NewListViewModel(session *app.Session) *ListViewModel {
	m := &ListViewModel{app: session, keys: keys.Standard.ListView, formKeys: keys.Standard.Form, shell: tui.NewListDetail("Audit", "Loading audit entries...", styles.Standard.ListView), detail: NewDetailViewModel(styles.Standard.ListView), query: auditQuery{scope: scopeAll, limit: paging.DefaultLimit}}
	m.shell.SetLocalFiltering(false)
	m.shell.SetLocalPagination(false)
	m.updateTitle()
	return m
}
func (m *ListViewModel) Init() tea.Cmd { return tea.Batch(m.shell.BeginLoading(), m.loadEntries()) }
func (m *ListViewModel) Interaction() tui.Interaction {
	return tui.Interaction{CapturesText: m.filter != nil, HandlesBack: m.filter != nil}
}
func (m *ListViewModel) Update(message tea.Msg) (tui.ViewModel, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		if m.filter != nil {
			m.filter.SetSize(msg.Width, msg.Height)
		}
		return m, nil
	case tea.KeyMsg:
		if m.filter != nil {
			if key.Matches(msg, m.keys.Back) && !m.filter.form.IsEditing() {
				m.filter = nil
				return m, nil
			}
			if key.Matches(msg, m.formKeys.Submit) {
				q, _, err := m.filter.Query()
				if err != nil {
					return m, nil
				}
				m.query = q
				m.next = ""
				m.history = nil
				m.filter = nil
				m.updateTitle()
				return m, tea.Batch(m.shell.BeginLoading(), m.loadEntries())
			}
			return m, m.filter.Update(msg)
		}
		switch {
		case key.Matches(msg, m.keys.Refresh):
			return m, tea.Batch(m.shell.BeginLoading(), m.loadEntries())
		case msg.String() == "f":
			m.filter = newFilterVM(m.query)
			m.filter.SetSize(m.width, m.height)
			return m, m.filter.Init()
		case msg.String() == "]" && m.next != "":
			m.history = append(m.history, m.query.cursor)
			m.query.cursor = m.next
			return m, tea.Batch(m.shell.BeginLoading(), m.loadEntries())
		case msg.String() == "[" && len(m.history) > 0:
			i := len(m.history) - 1
			m.query.cursor = m.history[i]
			m.history = m.history[:i]
			return m, tea.Batch(m.shell.BeginLoading(), m.loadEntries())
		}
	case AuditLoadedMsg:
		if msg.Token != m.loadToken {
			return m, nil
		}
		if msg.Err != nil {
			// A failed refresh must not destroy the last successful result or its
			// selection. This matches the other list surfaces and keeps the error
			// recoverable with another refresh.
			m.shell.SetResult(m.shell.Items(), msg.Err)
			return m, nil
		}
		selected := selectedAuditID(m.shell.SelectedItem())
		items := make([]list.Item, 0, len(msg.Entries))
		for _, entry := range msg.Entries {
			items = append(items, newAuditItem(entry))
		}
		m.next = msg.Next
		m.shell.SetResult(items, msg.Err)
		selectAuditID(m.shell, selected)
		m.syncDetail()
		m.updateTitle()
		return m, nil
	}
	cmd := m.shell.Update(message)
	m.syncDetail()
	return m, cmd
}
func (m *ListViewModel) View() string {
	if m.filter != nil {
		return m.filter.View()
	}
	return m.shell.View(m.detail.View())
}
func (m *ListViewModel) ShortHelp() []key.Binding {
	return []key.Binding{m.keys.Up, m.keys.Down, m.shell.KeyMap().PrevPage, m.shell.KeyMap().NextPage, m.keys.Refresh, m.keys.Back}
}
func (m *ListViewModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{{m.keys.Up, m.keys.Down, m.keys.Enter}, {m.shell.KeyMap().PrevPage, m.shell.KeyMap().NextPage}, {m.keys.Refresh, m.keys.Back}}
}
func (m *ListViewModel) loadEntries() tea.Cmd {
	m.loadToken++
	token := m.loadToken
	q := m.query
	return func() tea.Msg {
		req, err := q.Request()
		if err != nil {
			return AuditLoadedMsg{Err: err, Token: token}
		}
		page, err := m.app.Audit.List(m.context(), req)
		if err != nil {
			return AuditLoadedMsg{Err: err, Token: token}
		}
		rows := make([]auditmodels.AuditEntry, 0, len(page.Items))
		for i, entry := range page.Items {
			if entry == nil {
				return AuditLoadedMsg{Err: errors.Internalf("audit entry %d missing", i), Token: token}
			}
			rows = append(rows, *entry)
		}
		return AuditLoadedMsg{Entries: rows, Next: page.Next, Token: token}
	}
}
func (m *ListViewModel) setSize(width, height int) {
	m.width, m.height = width, height
	_, detailWidth := m.shell.SetSize(width, height)
	m.detail.SetSize(detailWidth, height)
}
func (m *ListViewModel) syncDetail() {
	item, ok := m.shell.SelectedItem().(auditItem)
	if !ok {
		m.detail.SetEntry(optional.None[auditmodels.AuditEntry]())
		return
	}
	m.detail.SetEntry(optional.Some(item.Value))
}
func (m *ListViewModel) context() *middleware.Context { return m.app.Context() }
func (m *ListViewModel) updateTitle() {
	title := fmt.Sprintf("Audit · %s · page %d", m.query.scope, len(m.history)+1)
	m.shell.SetTitle(title)
}
func selectedAuditID(item list.Item) string {
	if row, ok := item.(auditItem); ok {
		return row.Value.ID.String()
	}
	return ""
}
func selectAuditID(shell *tui.ListDetail, id string) {
	if id == "" {
		return
	}
	for i, item := range shell.Items() {
		if selectedAuditID(item) == id {
			shell.Select(i)
			return
		}
	}
}
