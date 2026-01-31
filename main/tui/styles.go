package tui

import "github.com/charmbracelet/lipgloss"

// Styles holds all the Lip Gloss styles used in the TUI.
type Styles struct {
	// Colors
	Primary   lipgloss.AdaptiveColor
	Secondary lipgloss.AdaptiveColor
	Success   lipgloss.AdaptiveColor
	Warning   lipgloss.AdaptiveColor
	Error     lipgloss.AdaptiveColor
	Muted     lipgloss.AdaptiveColor

	// Component styles
	Title      lipgloss.Style
	Subtitle   lipgloss.Style
	Selected   lipgloss.Style
	Unselected lipgloss.Style
	StatusBar  lipgloss.Style
	ErrorText  lipgloss.Style
	HelpKey    lipgloss.Style
	HelpDesc   lipgloss.Style

	// Layout styles
	Border        lipgloss.Style
	FocusedBorder lipgloss.Style
	Card          lipgloss.Style
}

// NewStyles creates a Styles instance with the default theme.
func NewStyles() Styles {
	styles := Styles{
		Primary:   lipgloss.AdaptiveColor{Light: "#1d4ed8", Dark: "#7aa2f7"},
		Secondary: lipgloss.AdaptiveColor{Light: "#0f766e", Dark: "#2dd4bf"},
		Success:   lipgloss.AdaptiveColor{Light: "#15803d", Dark: "#4ade80"},
		Warning:   lipgloss.AdaptiveColor{Light: "#b45309", Dark: "#fbbf24"},
		Error:     lipgloss.AdaptiveColor{Light: "#b91c1c", Dark: "#f87171"},
		Muted:     lipgloss.AdaptiveColor{Light: "#6b7280", Dark: "#9ca3af"},
	}

	selectedForeground := lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#0b1120"}
	statusForeground := lipgloss.AdaptiveColor{Light: "#f8fafc", Dark: "#e2e8f0"}
	statusBackground := lipgloss.AdaptiveColor{Light: "#0f172a", Dark: "#111827"}

	styles.Title = lipgloss.NewStyle().Bold(true).Foreground(styles.Primary)
	styles.Subtitle = lipgloss.NewStyle().Foreground(styles.Secondary)
	styles.Selected = lipgloss.NewStyle().
		Bold(true).
		Foreground(selectedForeground).
		Background(styles.Primary)
	styles.Unselected = lipgloss.NewStyle().Foreground(styles.Muted)
	styles.StatusBar = lipgloss.NewStyle().
		Foreground(statusForeground).
		Background(statusBackground).
		Padding(0, 1)
	styles.ErrorText = lipgloss.NewStyle().Bold(true).Foreground(styles.Error)
	styles.HelpKey = lipgloss.NewStyle().Bold(true).Foreground(styles.Primary)
	styles.HelpDesc = lipgloss.NewStyle().Foreground(styles.Muted)

	styles.Border = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted)
	styles.FocusedBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary)
	styles.Card = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(1, 2)

	return styles
}
