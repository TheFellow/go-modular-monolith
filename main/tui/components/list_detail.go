package components

import (
	"fmt"

	sharedtui "github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ListDetail is the standard presentation shell for searchable list/detail
// screens. Domain ViewModels retain loading commands, typed selection, detail
// rendering, and workflows; this component owns the recurring terminal state.
type ListDetail struct {
	styles  sharedtui.ListViewStyles
	list    list.Model
	spinner Spinner
	loading bool
	err     error
	width   int
	height  int
	listW   int
	detailW int
}

func NewListDetail(title, loadingLabel string, styles sharedtui.ListViewStyles) *ListDetail {
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

	listWidth, detailWidth = sharedtui.SplitListDetailWidths(width)
	listWidth = sharedtui.PaneContentWidth(m.styles.ListPane, listWidth)
	detailWidth = sharedtui.PaneContentWidth(m.styles.DetailPane, detailWidth)
	m.list.SetSize(listWidth, height)
	m.listW, m.detailW = listWidth, detailWidth
	return listWidth, detailWidth
}

func (m *ListDetail) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	if m.loading {
		m.spinner, cmd = m.spinner.Update(msg)
		return cmd
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
	listView = m.styles.ListPane.Width(sharedtui.PaneStyleWidth(m.styles.ListPane, m.listW)).Render(listView)
	detail = m.styles.DetailPane.Width(sharedtui.PaneStyleWidth(m.styles.DetailPane, m.detailW)).Render(detail)
	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detail)
}

func (m *ListDetail) Filtering() bool         { return m.list.SettingFilter() }
func (m *ListDetail) Loading() bool           { return m.loading }
func (m *ListDetail) SelectedItem() list.Item { return m.list.SelectedItem() }
func (m *ListDetail) KeyMap() list.KeyMap     { return m.list.KeyMap }
