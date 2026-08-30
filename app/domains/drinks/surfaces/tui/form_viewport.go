package tui

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
)

const recipeNavigationHelp = "↑/↓: recipe field • ←/→: choices\nenter: choose/toggle • tab: next form field"

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
