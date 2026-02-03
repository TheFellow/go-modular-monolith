package tui

import (
	"context"
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app"
	auditmodels "github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/queries"
	"github.com/TheFellow/go-modular-monolith/main/tui/components"
	tuikeys "github.com/TheFellow/go-modular-monolith/main/tui/keys"
	tuistyles "github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/main/tui/views"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/cedar-policy/cedar-go"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const auditDefaultLimit = 50

// ListViewModel renders the audit list and detail panes.
type ListViewModel struct {
	app       *app.App
	principal cedar.EntityUID
	styles    tui.ListViewStyles
	keys      tui.ListViewKeys

	queries *queries.Queries

	list    list.Model
	detail  *DetailViewModel
	spinner components.Spinner
	loading bool
	err     error

	width       int
	height      int
	listWidth   int
	detailWidth int
}

func NewListViewModel(app *app.App, principal cedar.EntityUID) *ListViewModel {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = tuistyles.App.ListView.Selected
	delegate.Styles.SelectedDesc = tuistyles.App.ListView.Selected

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Audit"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(true)

	vm := &ListViewModel{
		app:       app,
		principal: principal,
		styles:    tuistyles.App.ListView,
		keys:      tuikeys.App.ListView,
		queries:   queries.New(),
		list:      l,
		detail:    NewDetailViewModel(tuistyles.App.ListView),
		loading:   true,
	}
	vm.spinner = components.NewSpinner("Loading audit entries...", vm.styles.Subtitle)
	return vm
}

func (m *ListViewModel) Init() tea.Cmd {
	m.loading = true
	return tea.Batch(m.spinner.Init(), m.loadEntries())
}

func (m *ListViewModel) Update(msg tea.Msg) (views.ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Refresh):
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Init(), m.loadEntries())
		}
	case AuditLoadedMsg:
		m.loading = false
		m.err = msg.Err
		items := make([]list.Item, 0, len(msg.Entries))
		for _, entry := range msg.Entries {
			items = append(items, newAuditItem(entry))
		}
		m.list.SetItems(items)
		m.syncDetail()
		return m, nil
	}

	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.syncDetail()
	return m, cmd
}

func (m *ListViewModel) View() string {
	if m.loading {
		return m.renderLoading()
	}

	listView := m.list.View()
	if m.err != nil {
		listView = m.styles.ErrorText.Render(fmt.Sprintf("Error: %v", m.err))
	}
	listView = m.styles.ListPane.Width(m.listWidth).Render(listView)

	detailView := m.detail.View()
	detailView = m.styles.DetailPane.Width(m.detailWidth).Render(detailView)

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)
}

func (m *ListViewModel) ShortHelp() []key.Binding {
	return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Refresh, m.keys.Back}
}

func (m *ListViewModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.keys.Up, m.keys.Down, m.keys.Enter},
		{m.keys.Refresh, m.keys.Back},
	}
}

func (m *ListViewModel) loadEntries() tea.Cmd {
	return func() tea.Msg {
		entries, err := m.queries.List(m.context(), queries.ListFilter{Limit: auditDefaultLimit})
		if err != nil {
			return AuditLoadedMsg{Err: err}
		}

		rows := make([]auditmodels.AuditEntry, 0, len(entries))
		for i, entry := range entries {
			if entry == nil {
				return AuditLoadedMsg{Err: errors.Internalf("audit entry %d missing", i)}
			}
			rows = append(rows, *entry)
		}

		return AuditLoadedMsg{Entries: rows}
	}
}

func (m *ListViewModel) renderLoading() string {
	content := m.spinner.View()
	if m.width <= 0 || m.height <= 0 {
		return content
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m *ListViewModel) setSize(width, height int) {
	m.width = width
	m.height = height

	if width <= 0 {
		return
	}

	listWidth, detailWidth := views.SplitListDetailWidths(width)

	m.list.SetSize(listWidth, height)
	m.detail.SetSize(detailWidth, height)
	m.listWidth = listWidth
	m.detailWidth = detailWidth
}

func (m *ListViewModel) syncDetail() {
	item, ok := m.list.SelectedItem().(auditItem)
	if !ok {
		m.detail.SetEntry(optional.None[auditmodels.AuditEntry]())
		return
	}
	m.detail.SetEntry(optional.Some(item.entry))
}

func (m *ListViewModel) context() *middleware.Context {
	return m.app.Context(context.Background(), m.principal)
}
