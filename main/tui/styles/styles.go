package styles

import (
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/dialog"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	"github.com/charmbracelet/lipgloss"
)

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
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Selected    lipgloss.Style
	Unselected  lipgloss.Style
	StatusBar   lipgloss.Style
	ErrorText   lipgloss.Style
	WarningText lipgloss.Style
	InfoText    lipgloss.Style
	HelpKey     lipgloss.Style
	HelpDesc    lipgloss.Style
	TitleBar    lipgloss.Style

	// Form styles
	FormLabel         lipgloss.Style
	FormLabelRequired lipgloss.Style
	FormInput         lipgloss.Style
	FormInputFocused  lipgloss.Style
	FormError         lipgloss.Style
	FormHelp          lipgloss.Style

	// Dialog styles
	DialogModal       lipgloss.Style
	DialogTitle       lipgloss.Style
	DialogMessage     lipgloss.Style
	DialogButton      lipgloss.Style
	DialogButtonFocus lipgloss.Style
	DialogDanger      lipgloss.Style

	// Layout styles
	Border        lipgloss.Style
	FocusedBorder lipgloss.Style
	Card          lipgloss.Style
	ListPane      lipgloss.Style
	DetailPane    lipgloss.Style

	// Derived subsets
	ListView  tui.ListViewStyles
	Form      forms.FormStyles
	Dialog    dialog.DialogStyles
	Dashboard DashboardStyles
}

// App is the shared application style set.
var App = newStyles()

// DashboardStyles contains the lipgloss styles used by the dashboard.
type DashboardStyles struct {
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Card     lipgloss.Style
	HelpKey  lipgloss.Style
}

// newStyles creates a Styles instance with the default theme.
func newStyles() Styles {
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
	styles.WarningText = lipgloss.NewStyle().Bold(true).Foreground(styles.Warning)
	styles.InfoText = lipgloss.NewStyle().Foreground(styles.Muted)
	styles.HelpKey = lipgloss.NewStyle().Bold(true).Foreground(styles.Primary)
	styles.HelpDesc = lipgloss.NewStyle().Foreground(styles.Muted)
	styles.TitleBar = lipgloss.NewStyle().
		Bold(true).
		Foreground(statusForeground).
		Background(styles.Primary).
		Padding(0, 1).
		MarginBottom(1)

	styles.FormLabel = lipgloss.NewStyle().Foreground(styles.Muted)
	styles.FormLabelRequired = lipgloss.NewStyle().Bold(true).Foreground(styles.Error)
	styles.FormInput = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#0f172a", Dark: "#e2e8f0"})
	styles.FormInputFocused = lipgloss.NewStyle().Bold(true).Foreground(styles.Primary)
	styles.FormError = styles.ErrorText
	styles.FormHelp = styles.InfoText

	styles.DialogModal = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 2)
	styles.DialogTitle = styles.Title
	styles.DialogMessage = lipgloss.NewStyle().Foreground(styles.Muted)
	styles.DialogButton = lipgloss.NewStyle().
		Foreground(styles.Primary).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(0, 1)
	styles.DialogButtonFocus = lipgloss.NewStyle().Bold(true).Underline(true)
	styles.DialogDanger = lipgloss.NewStyle().Bold(true).Foreground(styles.Error)

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
	styles.ListPane = lipgloss.NewStyle().
		Width(60).
		Padding(0, 1)
	styles.DetailPane = lipgloss.NewStyle().
		Width(40).
		Padding(0, 1).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.Muted)

	styles.ListView = listViewStylesFrom(styles)
	styles.Form = formStylesFrom(styles)
	styles.Dialog = dialogStylesFrom(styles)
	styles.Dashboard = dashboardStylesFrom(styles)

	return styles
}

func listViewStylesFrom(s Styles) tui.ListViewStyles {
	return tui.ListViewStyles{
		Title:       s.Title,
		Subtitle:    s.Subtitle,
		Muted:       s.Unselected,
		Selected:    s.Selected,
		ListPane:    s.ListPane,
		DetailPane:  s.DetailPane,
		ErrorText:   s.ErrorText,
		WarningText: s.WarningText,
	}
}

func formStylesFrom(s Styles) forms.FormStyles {
	return forms.FormStyles{
		Form:          lipgloss.NewStyle(),
		Label:         s.FormLabel,
		LabelRequired: s.FormLabelRequired,
		Input:         s.FormInput,
		InputFocused:  s.FormInputFocused,
		Error:         s.FormError,
		Help:          s.FormHelp,
	}
}

func dialogStylesFrom(s Styles) dialog.DialogStyles {
	return dialog.DialogStyles{
		Modal:         s.DialogModal,
		Title:         s.DialogTitle,
		Message:       s.DialogMessage,
		Button:        s.DialogButton,
		ButtonFocused: s.DialogButtonFocus,
		DangerButton:  s.DialogDanger,
	}
}

func dashboardStylesFrom(s Styles) DashboardStyles {
	return DashboardStyles{
		Title:    s.Title,
		Subtitle: s.Subtitle,
		Card:     s.Card,
		HelpKey:  s.HelpKey,
	}
}
