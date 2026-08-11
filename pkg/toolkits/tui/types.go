package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// DataInvalidatedMsg is a coalesced hint that persisted state may have
// changed. View models respond by running their ordinary authorized load path.
type DataInvalidatedMsg struct{ Epoch uint64 }

// ListViewStyles contains styles needed by domain list ViewModels.
type ListViewStyles struct {
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Muted       lipgloss.Style
	Selected    lipgloss.Style
	ListPane    lipgloss.Style
	DetailPane  lipgloss.Style
	ErrorText   lipgloss.Style
	WarningText lipgloss.Style
}
