package tui

import (
	"strings"
	"unicode"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/main/tui/components"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/charmbracelet/lipgloss"
)

func menuStatusLabel(status models.MenuStatus) string {
	switch status {
	case models.MenuStatusDraft:
		return "Draft"
	case models.MenuStatusPublished:
		return "Published"
	case models.MenuStatusArchived:
		return "Archived"
	default:
		return titleCase(string(status))
	}
}

func menuStatusStyle(status models.MenuStatus, styles tui.ListViewStyles) lipgloss.Style {
	switch status {
	case models.MenuStatusDraft:
		return styles.WarningText
	case models.MenuStatusPublished:
		return styles.Subtitle
	case models.MenuStatusArchived:
		return styles.Muted
	default:
		return styles.Muted
	}
}

func menuStatusBadge(status models.MenuStatus, styles tui.ListViewStyles) string {
	label := menuStatusLabel(status)
	style := menuStatusStyle(status, styles)
	return components.NewBadge(label, style).View()
}

func titleCase(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	runes := []rune(value)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
