package tui

import (
	"strings"
	"unicode"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/main/tui/components"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/charmbracelet/lipgloss"
)

func orderStatusLabel(status models.OrderStatus) string {
	switch status {
	case models.OrderStatusPending:
		return "Pending"
	case models.OrderStatusPreparing:
		return "Preparing"
	case models.OrderStatusCompleted:
		return "Completed"
	case models.OrderStatusCancelled:
		return "Cancelled"
	default:
		return titleCase(string(status))
	}
}

func orderStatusStyle(status models.OrderStatus, styles tui.ListViewStyles) lipgloss.Style {
	switch status {
	case models.OrderStatusPending:
		return styles.WarningText
	case models.OrderStatusPreparing:
		return styles.Subtitle
	case models.OrderStatusCompleted:
		return styles.Subtitle
	case models.OrderStatusCancelled:
		return styles.ErrorText
	default:
		return styles.Muted
	}
}

func orderStatusBadge(status models.OrderStatus, styles tui.ListViewStyles) string {
	label := orderStatusLabel(status)
	style := orderStatusStyle(status, styles)
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
