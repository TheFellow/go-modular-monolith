package gui

import "errors"

import apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"

// ErrorSeverity is deliberately neutral presentation policy. A bespoke
// surface decides whether and where a Presentation is rendered.
type ErrorSeverity uint8

const (
	ErrorSeverityInline ErrorSeverity = iota
	ErrorSeverityWarning
	ErrorSeverityError
)

// ErrorPresentation contains only text safe to display to an end user. Cause
// remains available for errors.Is/errors.As without exposing diagnostic detail.
type ErrorPresentation struct {
	Severity ErrorSeverity
	Message  string
	Cause    error
}

func (p ErrorPresentation) Error() string { return p.Message }
func (p ErrorPresentation) Unwrap() error { return p.Cause }

// PresentError maps application error kinds into desktop-neutral severity and
// safe copy. Unknown errors are treated as internal failures.
func PresentError(err error) error {
	if err == nil {
		return nil
	}
	presentation := ErrorPresentation{Severity: ErrorSeverityError, Message: apperrors.SpecFor(apperrors.KindInternal).Message, Cause: err}
	var typed *apperrors.Error
	if !apperrors.As(err, &typed) {
		return presentation
	}
	presentation.Message = typed.UserMessage()
	switch typed.Kind() {
	case apperrors.KindInvalid:
		presentation.Severity = ErrorSeverityInline
	case apperrors.KindNotFound, apperrors.KindConflict, apperrors.KindFailedPrecondition:
		presentation.Severity = ErrorSeverityWarning
	case apperrors.KindPermission, apperrors.KindInternal:
		presentation.Severity = ErrorSeverityError
	default:
		presentation.Message = apperrors.SpecFor(apperrors.KindInternal).Message
	}
	return presentation
}

// ShowPresentation renders non-inline failures through injected dialogs.
// Validation remains visible in the active form and can be corrected in place.
func ShowPresentation(dialogs Dialogs, err error) {
	if dialogs == nil || err == nil {
		return
	}
	var presented ErrorPresentation
	if !errors.As(PresentError(err), &presented) {
		return
	}
	switch presented.Severity {
	case ErrorSeverityInline:
		return
	case ErrorSeverityWarning:
		dialogs.ShowWarning("Unable to complete operation", presented.Message)
	case ErrorSeverityError:
		dialogs.ShowError(presented)
	}
}
