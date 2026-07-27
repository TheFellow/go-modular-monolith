package tui

import (
	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	auditmodels "github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	tuikeys "github.com/TheFellow/go-modular-monolith/main/tui/keys"
	tuistyles "github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/main/tui/views"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

const auditDefaultLimit = 50

// ListViewModel renders the audit list and detail panes.
type ListViewModel struct {
	app  *app.Session
	keys tui.ListViewKeys

	shell  *tui.ListDetail
	detail *DetailViewModel
}

func NewListViewModel(app *app.Session) *ListViewModel {
	return &ListViewModel{
		app:    app,
		keys:   tuikeys.App.ListView,
		shell:  tui.NewListDetail("Audit", "Loading audit entries...", tuistyles.App.ListView),
		detail: NewDetailViewModel(tuistyles.App.ListView),
	}
}

func (m *ListViewModel) Init() tea.Cmd {
	return tea.Batch(m.shell.BeginLoading(), m.loadEntries())
}

func (m *ListViewModel) Interaction() views.Interaction {
	filtering := m.shell.Filtering()
	return views.Interaction{CapturesText: filtering, HandlesBack: filtering}
}

func (m *ListViewModel) Update(msg tea.Msg) (views.ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		if m.shell.Filtering() {
			break
		}
		if key.Matches(msg, m.keys.Refresh) {
			return m, tea.Batch(m.shell.BeginLoading(), m.loadEntries())
		}
	case AuditLoadedMsg:
		items := make([]list.Item, 0, len(msg.Entries))
		for _, entry := range msg.Entries {
			items = append(items, newAuditItem(entry))
		}
		m.shell.SetResult(items, msg.Err)
		m.syncDetail()
		return m, nil
	}

	cmd := m.shell.Update(msg)
	m.syncDetail()
	return m, cmd
}

func (m *ListViewModel) View() string {
	return m.shell.View(m.detail.View())
}

func (m *ListViewModel) ShortHelp() []key.Binding {
	return []key.Binding{
		m.keys.Up, m.keys.Down,
		m.shell.KeyMap().PrevPage, m.shell.KeyMap().NextPage,
		m.keys.Refresh, m.keys.Back,
	}
}

func (m *ListViewModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.keys.Up, m.keys.Down, m.keys.Enter},
		{m.shell.KeyMap().PrevPage, m.shell.KeyMap().NextPage},
		{m.keys.Refresh, m.keys.Back},
	}
}

func (m *ListViewModel) loadEntries() tea.Cmd {
	return func() tea.Msg {
		entries, err := m.app.Audit.List(m.context(), audit.ListRequest{Limit: auditDefaultLimit})
		if err != nil {
			return AuditLoadedMsg{Err: err}
		}

		rows := make([]auditmodels.AuditEntry, 0, len(entries.Items))
		for i, entry := range entries.Items {
			if entry == nil {
				return AuditLoadedMsg{Err: errors.Internalf("audit entry %d missing", i)}
			}
			rows = append(rows, *entry)
		}

		return AuditLoadedMsg{Entries: rows}
	}
}

func (m *ListViewModel) setSize(width, height int) {
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

func (m *ListViewModel) context() *middleware.Context {
	return m.app.Context()
}
