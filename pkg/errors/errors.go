package errors

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
)

type httpCode int

type ErrorKind struct {
	Name     string
	Message  string
	HTTPCode httpCode
	GRPCCode codes.Code
	CLICode  int
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
//	50 - Internal error
const (
	ExitSuccess    = 0
	ExitGeneral    = 1
	ExitUsage      = 2
	ExitInvalid    = 10
	ExitNotFound   = 20
	ExitPermission = 30
	ExitInternal   = 50
)

var (
	ErrInvalid = ErrorKind{
		Name:     "Invalid",
		Message:  "invalid",
		HTTPCode: http.StatusBadRequest,
		GRPCCode: codes.InvalidArgument,
		CLICode:  ExitInvalid,
	}
	ErrNotFound = ErrorKind{
		Name:     "NotFound",
		Message:  "not found",
		HTTPCode: http.StatusNotFound,
		GRPCCode: codes.NotFound,
		CLICode:  ExitNotFound,
	}
	ErrPermission = ErrorKind{
		Name:     "Permission",
		Message:  "permission denied",
		HTTPCode: http.StatusForbidden,
		GRPCCode: codes.PermissionDenied,
		CLICode:  ExitPermission,
	}
	ErrInternal = ErrorKind{
		Name:     "Internal",
		Message:  "internal error",
		HTTPCode: http.StatusInternalServerError,
		GRPCCode: codes.Internal,
		CLICode:  ExitInternal,
	}

	ErrorKinds = []ErrorKind{ErrInvalid, ErrNotFound, ErrPermission, ErrInternal}
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
