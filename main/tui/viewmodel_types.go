package tui

import (
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/dialog"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	"github.com/charmbracelet/lipgloss"
)

// ListViewStylesFrom creates ListViewStyles from the main Styles.
func ListViewStylesFrom(s Styles) tui.ListViewStyles {
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

// ListViewKeysFrom creates ListViewKeys from the main KeyMap.
func ListViewKeysFrom(k KeyMap) tui.ListViewKeys {
	return tui.ListViewKeys{
		Up:      k.Up,
		Down:    k.Down,
		Enter:   k.Enter,
		Refresh: k.Refresh,
		Back:    k.Back,
		Create:  k.Create,
		Edit:    k.Edit,
		Delete:  k.Delete,
		Adjust:  k.Adjust,
		Set:     k.Set,
		Publish: k.Publish,
	}
}

// FormStylesFrom creates FormStyles from the main Styles.
func FormStylesFrom(s Styles) forms.FormStyles {
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

// FormKeysFrom creates FormKeys from the main KeyMap.
func FormKeysFrom(k KeyMap) forms.FormKeys {
	return forms.FormKeys{
		NextField: k.NextField,
		PrevField: k.PrevField,
		Submit:    k.Submit,
		Cancel:    k.Back,
	}
}

// DialogStylesFrom creates DialogStyles from the main Styles.
func DialogStylesFrom(s Styles) dialog.DialogStyles {
	return dialog.DialogStyles{
		Modal:         s.DialogModal,
		Title:         s.DialogTitle,
		Message:       s.DialogMessage,
		Button:        s.DialogButton,
		ButtonFocused: s.DialogButtonFocus,
		DangerButton:  s.DialogDanger,
	}
}

// DialogKeysFrom creates DialogKeys from the main KeyMap.
func DialogKeysFrom(k KeyMap) dialog.DialogKeys {
	return dialog.DialogKeys{
		Confirm: k.Confirm,
		Cancel:  k.Back,
		Switch:  k.SwitchBtn,
	}
}
