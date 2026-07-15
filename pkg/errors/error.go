//go:generate go run ./gen

package errors

import (
	stderrors "errors"
	"fmt"

	"google.golang.org/grpc/codes"
)

// Error is the shared payload behind each generated typed error. Detail is
// useful for logs and diagnostics; userMessage is safe for presentation.
type Error struct {
	kind        Kind
	detail      string
	userMessage string
	cause       error
}

func newErrorf(kind Kind, format string, args ...any) *Error {
	if format == "" {
		return &Error{kind: kind}
	}

	formatted := fmt.Errorf(format, args...)
	return &Error{
		kind:   kind,
		detail: formatted.Error(),
		cause:  stderrors.Unwrap(formatted),
	}
}

func (e *Error) Error() string {
	if e == nil {
		return SpecFor(KindInternal).Message
	}
	if e.detail != "" {
		return e.detail
	}
	return SpecFor(e.kind).Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func (e *Error) Kind() Kind {
	if e == nil {
		return KindInternal
	}
	return e.kind
}

func (e *Error) HTTPStatus() int      { return e.Kind().HTTPStatus() }
func (e *Error) GRPCCode() codes.Code { return e.Kind().GRPCCode() }
func (e *Error) CLIExitCode() int     { return e.Kind().CLIExitCode() }
func (e *Error) TUIStyle() TUIStyle   { return e.Kind().TUIStyle() }

// UserMessage returns presentation-safe text. Internal errors default to a
// generic message; other kinds retain their actionable detail unless callers
// provide an explicit override.
func (e *Error) UserMessage() string {
	if e == nil {
		return SpecFor(KindInternal).Message
	}
	if e.userMessage != "" {
		return e.userMessage
	}
	if e.kind == KindInternal {
		return SpecFor(e.kind).Message
	}
	return e.Error()
}

func (e *Error) WithUserMessage(message string) *Error {
	if e != nil {
		e.userMessage = message
	}
	return e
}
