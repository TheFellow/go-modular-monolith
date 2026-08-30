package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// DetailScrollHelp is the shared hint for scrolling the secondary pane while
// arrow keys remain dedicated to primary-list selection.
var DetailScrollHelp = key.NewBinding(
	key.WithKeys("pgup", "pgdown", "ctrl+u", "ctrl+d"),
	key.WithHelp("pg↑/pg↓", "detail"),
)

// DetailViewport clips long secondary-pane content to the terminal and gives
// it pager-style navigation without introducing a separate focus mode.
type DetailViewport struct {
	model   viewport.Model
	content string
	width   int
	height  int
}

func NewDetailViewport() DetailViewport {
	return DetailViewport{model: viewport.New(0, 0)}
}

func (v *DetailViewport) SetSize(width, height int) {
	v.width, v.height = max(width, 0), max(height, 0)
	v.model.Width, v.model.Height = v.width, v.height
	v.model.SetYOffset(v.model.YOffset)
}

// Update handles only pager keys. List arrows therefore retain their usual
// Bubble list/table behavior.
func (v *DetailViewport) Update(msg tea.Msg) bool {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok || v.height <= 0 {
		return false
	}
	switch keyMsg.String() {
	case "pgdown":
		v.model.PageDown()
	case "pgup":
		v.model.PageUp()
	case "ctrl+d":
		v.model.HalfPageDown()
	case "ctrl+u":
		v.model.HalfPageUp()
	default:
		return false
	}
	return true
}

func (v *DetailViewport) View(content string) string {
	if v.width <= 0 || v.height <= 0 {
		return content
	}
	if content != v.content {
		v.content = content
		v.model.SetContent(content)
		v.model.GotoTop()
	} else {
		v.model.SetContent(content)
	}
	return v.model.View()
}
