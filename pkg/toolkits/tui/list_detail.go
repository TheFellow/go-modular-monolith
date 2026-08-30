package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ListDetail owns the recurring terminal state for searchable list/detail
// screens. Callers retain loading commands, typed selection, detail rendering,
// and workflow state.
type ListDetail struct {
	styles  ListViewStyles
	list    list.Model
	spinner Spinner
	loading bool
	err     error
	width   int
	height  int
	listW   int
	detailW int
	detail  DetailViewport
}

func NewListDetail(title, loadingLabel string, styles ListViewStyles) *ListDetail {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = styles.Selected
	delegate.Styles.SelectedDesc = styles.Selected

	l := list.New(nil, delegate, 0, 0)
	l.Title = title
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(true)
	l.Paginator.Type = paginator.Arabic
	l.SetFilteringEnabled(true)

	return &ListDetail{
		styles:  styles,
		list:    l,
		spinner: NewSpinner(loadingLabel, styles.Subtitle),
		detail:  NewDetailViewport(),
		loading: true,
	}
}

// BeginLoading resets transient errors and returns the spinner command that
// should be batched with the domain load command.
func (m *ListDetail) BeginLoading() tea.Cmd {
	m.loading = true
	m.err = nil
	return m.spinner.Init()
}

func (m *ListDetail) SetResult(items []list.Item, err error) {
	m.loading = false
	m.err = err
	m.list.SetItems(items)
}

func (m *ListDetail) SetError(err error) { m.err = err }

func (m *ListDetail) SetSize(width, height int) (listWidth, detailWidth int) {
	m.width, m.height = width, height
	if width <= 0 {
		return 0, 0
	}

	listWidth, detailWidth = SplitListDetailWidths(width)
	listWidth = PaneContentWidth(m.styles.ListPane, listWidth)
	detailWidth = PaneContentWidth(m.styles.DetailPane, detailWidth)
	m.list.SetSize(listWidth, height)
	_, detailFrameHeight := m.styles.DetailPane.GetFrameSize()
	m.detail.SetSize(detailWidth, max(height-detailFrameHeight, 1))
	m.listW, m.detailW = listWidth, detailWidth
	return listWidth, detailWidth
}

func (m *ListDetail) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	if m.loading {
		m.spinner, cmd = m.spinner.Update(msg)
		return cmd
	}
	if m.detail.Update(msg) {
		return nil
	}
	m.list, cmd = m.list.Update(msg)
	return cmd
}

func (m *ListDetail) View(detail string) string {
	if m.loading {
		content := m.spinner.View()
		if m.width > 0 && m.height > 0 {
			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
		}
		return content
	}

	listView := m.list.View()
	if m.err != nil {
		listView = m.styles.ErrorText.Render(fmt.Sprintf("Error: %v", m.err))
	}
	listView = m.styles.ListPane.Width(PaneStyleWidth(m.styles.ListPane, m.listW)).Render(listView)
	detail = m.detail.View(detail)
	detail = m.styles.DetailPane.Width(PaneStyleWidth(m.styles.DetailPane, m.detailW)).Render(detail)
	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detail)
}

func (m *ListDetail) Filtering() bool         { return m.list.SettingFilter() }
func (m *ListDetail) Loading() bool           { return m.loading }
func (m *ListDetail) SelectedItem() list.Item { return m.list.SelectedItem() }
func (m *ListDetail) KeyMap() list.KeyMap     { return m.list.KeyMap }

// Items returns a copy of the currently rendered adapter items. It permits a
// domain surface to restore selection by stable identity after a refresh
// without exposing the underlying Bubbles model.
func (m *ListDetail) Items() []list.Item {
	return append([]list.Item(nil), m.list.Items()...)
}

// Select selects a rendered item by zero-based index.
func (m *ListDetail) Select(index int) { m.list.Select(index) }

// SetTitle updates the browse header with domain-owned query context.
func (m *ListDetail) SetTitle(title string) { m.list.Title = title }

// SetLocalFiltering controls the optional Bubbles in-memory quick filter.
func (m *ListDetail) SetLocalFiltering(enabled bool) { m.list.SetFilteringEnabled(enabled) }

// SetLocalPagination controls pagination within the current server page.
func (m *ListDetail) SetLocalPagination(enabled bool) { m.list.SetShowPagination(enabled) }
