package tui

import (
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// FormViewport keeps long terminal forms inside the available window and
// follows the active field. Navigation help remains pinned below the form.
type FormViewport struct {
	model  viewport.Model
	width  int
	height int
}

func NewFormViewport() FormViewport {
	return FormViewport{model: viewport.New(0, 0)}
}

func (v *FormViewport) SetSize(width, height int) {
	v.width, v.height = max(width, 0), max(height, 0)
	v.model.Width = v.width
	v.model.Height = v.height
}

func (v *FormViewport) SetWidth(width int) {
	v.SetSize(width, v.height)
}

func (v *FormViewport) YOffset() int { return v.model.YOffset }

func (v *FormViewport) View(content string, focusLine int, footer string) string {
	if v.height <= 0 {
		if footer != "" {
			return content + "\n" + footer
		}
		return content
	}
	footerHeight := 0
	if footer != "" {
		footerHeight = lipgloss.Height(footer)
	}
	v.model.Height = max(v.height-footerHeight, 1)
	v.model.SetContent(content)
	v.model.SetYOffset(max(focusLine-v.model.Height/2, 0))
	view := v.model.View()
	if footer != "" {
		view += "\n" + footer
	}
	return view
}
