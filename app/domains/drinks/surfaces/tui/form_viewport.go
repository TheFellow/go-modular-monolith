package tui

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

const recipeNavigationHelp = "ctrl+n/p: recipe control\nenter: choose/toggle • arrows: choices"

// formViewport keeps large, dynamic terminal forms usable without leaking
// scrolling policy into the generic forms package. Navigation help is pinned
// below the viewport while a composite recipe field owns focus.
type formViewport struct {
	model  viewport.Model
	width  int
	height int
}

func newFormViewport() formViewport {
	return formViewport{model: viewport.New(0, 0)}
}

func (v *formViewport) SetSize(width, height int) {
	v.width, v.height = max(width, 0), max(height, 0)
	v.model.Width = v.width
	v.model.Height = v.height
}

func (v *formViewport) View(content string, focusLine int, footer string) string {
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
	// Centering gives dynamic picker windows room on both sides and makes
	// reverse arrow navigation visibly track the highlighted candidate too.
	v.model.SetYOffset(max(focusLine-v.model.Height/2, 0))
	view := v.model.View()
	if footer != "" {
		view += "\n" + footer
	}
	return view
}

func recipeFocusLine(recipeOffset int, recipe *RecipeEditor) int {
	if recipe == nil || !recipe.IsFocused() {
		return 0
	}
	return recipeOffset + recipe.focusLine()
}

func recipeFieldOffset(errorView string, preceding ...forms.Field) int {
	parts := make([]string, 0, len(preceding))
	for _, field := range preceding {
		parts = append(parts, field.View())
	}
	prefix := strings.Join(parts, "\n\n") + "\n\n"
	if errorView != "" {
		prefix = errorView + "\n\n" + prefix
	}
	return strings.Count(prefix, "\n")
}

func lineCount(value string) int {
	if value == "" {
		return 0
	}
	return strings.Count(value, "\n") + 1
}
