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
}

var (
	ErrInvalid = ErrorKind{
		Name:     "Invalid",
		Message:  "invalid",
		HTTPCode: http.StatusBadRequest,
		GRPCCode: codes.InvalidArgument,
	}
	ErrNotFound = ErrorKind{
		Name:     "NotFound",
		Message:  "not found",
		HTTPCode: http.StatusNotFound,
		GRPCCode: codes.NotFound,
	}
	ErrInternal = ErrorKind{
		Name:     "Internal",
		Message:  "internal error",
		HTTPCode: http.StatusInternalServerError,
		GRPCCode: codes.Internal,
	}

	ErrorKinds = []ErrorKind{ErrInvalid, ErrNotFound, ErrInternal}
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
	if cause != nil && msg == cause.Error() {
		return msg, nil
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
