package components

import "github.com/charmbracelet/lipgloss"

// Badge displays a styled status indicator.
type Badge struct {
	text  string
	style lipgloss.Style
}

func NewBadge(text string, style lipgloss.Style) Badge {
	return Badge{
		text:  text,
		style: style.Padding(0, 1),
	}
}

func (b Badge) View() string {
	return b.style.Render(b.text)
}

// BadgeStyles holds predefined badge styles.
type BadgeStyles struct {
	Draft     lipgloss.Style
	Published lipgloss.Style
	Pending   lipgloss.Style
	Completed lipgloss.Style
	Cancelled lipgloss.Style
	OK        lipgloss.Style
	Low       lipgloss.Style
	Out       lipgloss.Style
}

func NewBadgeStyles(primary, success, warning, error_ lipgloss.AdaptiveColor) BadgeStyles {
	return BadgeStyles{
		Draft:     lipgloss.NewStyle().Foreground(warning),
		Published: lipgloss.NewStyle().Foreground(success),
		Pending:   lipgloss.NewStyle().Foreground(warning),
		Completed: lipgloss.NewStyle().Foreground(success),
		Cancelled: lipgloss.NewStyle().Foreground(error_),
		OK:        lipgloss.NewStyle().Foreground(success),
		Low:       lipgloss.NewStyle().Foreground(warning),
		Out:       lipgloss.NewStyle().Foreground(error_),
	}
}
