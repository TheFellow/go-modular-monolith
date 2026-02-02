# Task 003: Main TUI Form Styles and Keys

## Goal

Add form-specific styles and keys to `main/tui/` and create conversion functions similar to `ListViewStylesFrom()`.

## Files to Modify

- `main/tui/styles.go` - Add form-related styles
- `main/tui/keys.go` - Add form-related key bindings
- `main/tui/viewmodel_types.go` - Add `FormStylesFrom()` and `FormKeysFrom()`

## Implementation

### Add to styles.go

```go
// main/tui/styles.go

// Add to Styles struct:
type Styles struct {
    // ... existing styles ...

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
}

// Update NewStyles() to initialize these
```

### Add to keys.go

```go
// main/tui/keys.go

// Add to KeyMap struct:
type KeyMap struct {
    // ... existing keys ...

    // Form keys
    NextField key.Binding  // tab
    PrevField key.Binding  // shift+tab
    Submit    key.Binding  // ctrl+s
    // Cancel already exists as Back/esc

    // Dialog keys
    Confirm   key.Binding  // enter
    // Cancel already exists as Back/esc
    SwitchBtn key.Binding  // tab
}
```

### Add to viewmodel_types.go

```go
// main/tui/viewmodel_types.go

import (
    "github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
    "github.com/TheFellow/go-modular-monolith/pkg/tui/dialog"
)

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

func FormKeysFrom(k KeyMap) forms.FormKeys {
    return forms.FormKeys{
        NextField: k.NextField,
        PrevField: k.PrevField,
        Submit:    k.Submit,
        Cancel:    k.Back,
    }
}

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

func DialogKeysFrom(k KeyMap) dialog.DialogKeys {
    return dialog.DialogKeys{
        Confirm: k.Confirm,
        Cancel:  k.Back,
        Switch:  k.SwitchBtn,
    }
}
```

## Notes

- Follow existing pattern from `ListViewStylesFrom()`
- Form styles should be consistent with existing TUI theme
- Error style should match existing `ErrorText` style
- Focused input should be visually distinct

## Checklist

- [ ] Add form styles to `main/tui/styles.go`
- [ ] Add dialog styles to `main/tui/styles.go`
- [ ] Add form keys to `main/tui/keys.go`
- [ ] Add `FormStylesFrom()` to `viewmodel_types.go`
- [ ] Add `FormKeysFrom()` to `viewmodel_types.go`
- [ ] Add `DialogStylesFrom()` to `viewmodel_types.go`
- [ ] Add `DialogKeysFrom()` to `viewmodel_types.go`
- [ ] `go build ./main/tui/...` passes
- [ ] `go test ./...` passes
