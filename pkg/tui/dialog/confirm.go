package dialog

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	defaultConfirmText = "Confirm"
	defaultCancelText  = "Cancel"
	maxDialogWidth     = 80
	minDialogWidth     = 24
)

// ConfirmDialog is a confirmation dialog with confirm/cancel actions.
type ConfirmDialog struct {
	title      string
	message    string
	confirmBtn string
	cancelBtn  string
	dangerous  bool
	focused    int
	confirmed  bool
	cancelled  bool
	styles     DialogStyles
	keys       DialogKeys
	width      int
}

// ConfirmMsg is emitted when the user confirms.
type ConfirmMsg struct{}

// CancelMsg is emitted when the user cancels.
type CancelMsg struct{}

// DialogOption configures a ConfirmDialog.
type DialogOption func(*ConfirmDialog)

// NewConfirmDialog creates a new ConfirmDialog.
func NewConfirmDialog(title, message string, opts ...DialogOption) *ConfirmDialog {
	dialog := &ConfirmDialog{
		title:      title,
		message:    message,
		confirmBtn: defaultConfirmText,
		cancelBtn:  defaultCancelText,
		focused:    0,
		styles:     defaultDialogStyles(),
		keys:       defaultDialogKeys(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(dialog)
		}
	}
	return dialog
}

// WithConfirmText sets the confirm button text.
func WithConfirmText(text string) DialogOption {
	return func(d *ConfirmDialog) {
		if strings.TrimSpace(text) == "" {
			return
		}
		d.confirmBtn = text
	}
}

// WithCancelText sets the cancel button text.
func WithCancelText(text string) DialogOption {
	return func(d *ConfirmDialog) {
		if strings.TrimSpace(text) == "" {
			return
		}
		d.cancelBtn = text
	}
}

// WithDangerous styles the confirm button as dangerous.
func WithDangerous() DialogOption {
	return func(d *ConfirmDialog) {
		d.dangerous = true
	}
}

// WithFocusCancel sets the initial focus to cancel.
func WithFocusCancel() DialogOption {
	return func(d *ConfirmDialog) {
		d.focused = 1
	}
}

// WithStyles sets custom dialog styles.
func WithStyles(styles DialogStyles) DialogOption {
	return func(d *ConfirmDialog) {
		d.styles = styles
	}
}

// WithKeys sets custom dialog key bindings.
func WithKeys(keys DialogKeys) DialogOption {
	return func(d *ConfirmDialog) {
		d.keys = keys
	}
}

// Init initializes the dialog.
func (d *ConfirmDialog) Init() tea.Cmd {
	return nil
}

// Update handles dialog input.
func (d *ConfirmDialog) Update(msg tea.Msg) (*ConfirmDialog, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		d.SetWidth(typed.Width)
		return d, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(typed, d.keys.Switch):
			d.toggleFocus()
			return d, nil
		case key.Matches(typed, d.keys.Cancel):
			d.cancelled = true
			d.confirmed = false
			return d, func() tea.Msg { return CancelMsg{} }
		case key.Matches(typed, d.keys.Confirm):
			if d.focused == 1 {
				d.cancelled = true
				d.confirmed = false
				return d, func() tea.Msg { return CancelMsg{} }
			}
			d.confirmed = true
			d.cancelled = false
			return d, func() tea.Msg { return ConfirmMsg{} }
		}
	}
	return d, nil
}

// View renders the dialog.
func (d *ConfirmDialog) View() string {
	contentWidth := d.dialogContentWidth()
	titleStyle := d.styles.Title.Copy().Width(contentWidth).Align(lipgloss.Center)
	messageStyle := d.styles.Message.Copy().Width(contentWidth)

	title := titleStyle.Render(d.title)
	message := messageStyle.Render(d.message)
	buttons := d.renderButtons()
	buttons = lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, buttons)

	body := strings.Join([]string{title, "", message, "", buttons}, "\n")
	modal := d.styles.Modal.Render(body)
	if d.width > 0 {
		return lipgloss.PlaceHorizontal(d.width, lipgloss.Center, modal)
	}
	return modal
}

// SetWidth sets the dialog width.
func (d *ConfirmDialog) SetWidth(w int) {
	d.width = w
}

// IsConfirmed returns true if confirm was selected.
func (d *ConfirmDialog) IsConfirmed() bool {
	return d.confirmed
}

// IsCancelled returns true if cancel was selected.
func (d *ConfirmDialog) IsCancelled() bool {
	return d.cancelled
}

func (d *ConfirmDialog) renderButtons() string {
	confirmLabel := "[" + d.confirmBtn + "]"
	cancelLabel := "[" + d.cancelBtn + "]"

	confirmStyle := d.styles.Button
	if d.dangerous {
		confirmStyle = d.styles.DangerButton
	}
	confirmView := confirmStyle.Render(confirmLabel)
	if d.focused == 0 {
		confirmView = d.styles.ButtonFocused.Render(confirmView)
	}

	cancelView := d.styles.Button.Render(cancelLabel)
	if d.focused == 1 {
		cancelView = d.styles.ButtonFocused.Render(cancelView)
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, confirmView, "  ", cancelView)
}

func (d *ConfirmDialog) toggleFocus() {
	if d.focused == 0 {
		d.focused = 1
		return
	}
	d.focused = 0
}

func (d *ConfirmDialog) dialogContentWidth() int {
	width := d.width
	if width <= 0 {
		width = maxDialogWidth
	}
	if width > maxDialogWidth {
		width = maxDialogWidth
	}
	if width < minDialogWidth {
		width = minDialogWidth
	}
	return width - 4
}

func defaultDialogStyles() DialogStyles {
	return DialogStyles{
		Modal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2),
		Title:         lipgloss.NewStyle().Bold(true),
		Message:       lipgloss.NewStyle(),
		Button:        lipgloss.NewStyle().Padding(0, 1),
		ButtonFocused: lipgloss.NewStyle().Bold(true).Underline(true),
		DangerButton:  lipgloss.NewStyle().Bold(true),
	}
}

func defaultDialogKeys() DialogKeys {
	return DialogKeys{
		Confirm: key.NewBinding(key.WithKeys("enter")),
		Cancel:  key.NewBinding(key.WithKeys("esc")),
		Switch:  key.NewBinding(key.WithKeys("tab", "left", "right")),
	}
}
