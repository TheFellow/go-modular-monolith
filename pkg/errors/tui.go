package errors

import "errors"

// TUIError represents a styled error for the terminal UI.
type TUIError struct {
	Style   TUIStyle
	Message string
	Err     error
}

type tuiStyler interface {
	TUIStyle() TUIStyle
}

// ToTUIError converts an error into a TUIError with a style.
func ToTUIError(err error) TUIError {
	if err == nil {
		return TUIError{Style: TUIStyleInfo}
	}

	style := TUIStyleError
	var styler tuiStyler
	if errors.As(err, &styler) {
		style = styler.TUIStyle()
	}

	return TUIError{
		Style:   style,
		Message: err.Error(),
		Err:     err,
	}
}
