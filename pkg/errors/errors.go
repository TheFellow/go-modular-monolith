//go:generate go run ./gen

package errors

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
)

type httpCode int
type TUIStyle int

const (
	TUIStyleError TUIStyle = iota
	TUIStyleWarning
	TUIStyleInfo
)

type ErrorKind struct {
	Name     string
	Message  string
	HTTPCode httpCode
	GRPCCode codes.Code
	CLICode  int
	TUIStyle TUIStyle
}

// Exit codes.
//
// Standard Unix conventions:
//
//	0 - Success
//	1 - General error
//	2 - Misuse of command (invalid args)
//
// Application-specific:
//
//	10 - Validation/invalid input error
//	20 - Not found error
//	40 - Conflict error
//	50 - Internal error
const (
	ExitSuccess    = 0
	ExitGeneral    = 1
	ExitUsage      = 2
	ExitInvalid    = 10
	ExitNotFound   = 20
	ExitPermission = 30
	ExitConflict   = 40
	ExitInternal   = 50
)

var (
	ErrInvalid = ErrorKind{
		Name:     "Invalid",
		Message:  "invalid",
		HTTPCode: http.StatusBadRequest,
		GRPCCode: codes.InvalidArgument,
		CLICode:  ExitInvalid,
		TUIStyle: TUIStyleError,
	}
	ErrNotFound = ErrorKind{
		Name:     "NotFound",
		Message:  "not found",
		HTTPCode: http.StatusNotFound,
		GRPCCode: codes.NotFound,
		CLICode:  ExitNotFound,
		TUIStyle: TUIStyleWarning,
	}
	ErrPermission = ErrorKind{
		Name:     "Permission",
		Message:  "permission denied",
		HTTPCode: http.StatusForbidden,
		GRPCCode: codes.PermissionDenied,
		CLICode:  ExitPermission,
		TUIStyle: TUIStyleError,
	}
	ErrConflict = ErrorKind{
		Name:     "Conflict",
		Message:  "conflict",
		HTTPCode: http.StatusConflict,
		GRPCCode: codes.AlreadyExists,
		CLICode:  ExitConflict,
		TUIStyle: TUIStyleWarning,
	}
	ErrInternal = ErrorKind{
		Name:     "Internal",
		Message:  "internal error",
		HTTPCode: http.StatusInternalServerError,
		GRPCCode: codes.Internal,
		CLICode:  ExitInternal,
		TUIStyle: TUIStyleError,
	}

	ErrorKinds = []ErrorKind{ErrInvalid, ErrNotFound, ErrPermission, ErrConflict, ErrInternal}
)

func formatf(format string, args ...any) (msg string, cause error) {
	if format == "" {
		return "", nil
	}

	if strings.Contains(format, "%w") {
		for _, arg := range args {
			if err, ok := arg.(error); ok {
				cause = err
			}
		}
		format = strings.ReplaceAll(format, "%w", "%v")
	}

	msg = fmt.Sprintf(format, args...)
	if msg == "" {
		return "", nil
	}

	if cause != nil {
		return msg, cause
	}

	return msg, nil
}

var (
	As     = errors.As
	New    = errors.New
	Unwrap = errors.Unwrap
)
