package errors

import (
	"net/http"
	"slices"

	"google.golang.org/grpc/codes"
)

// Kind classifies an application error independently of any transport.
type Kind uint8

const (
	KindInvalid Kind = iota
	KindNotFound
	KindPermission
	KindConflict
	KindFailedPrecondition
	KindInternal
	kindCount
)

// TUIStyle controls how a terminal surface presents an error.
type TUIStyle uint8

const (
	TUIStyleError TUIStyle = iota
	TUIStyleWarning
	TUIStyleInfo
)

// Spec describes how an error kind is represented across transports.
// Values returned by SpecFor are copies and cannot mutate the package mapping.
type Spec struct {
	Kind        Kind
	Name        string
	Message     string
	HTTPStatus  int
	GRPCCode    codes.Code
	CLIExitCode int
	TUIStyle    TUIStyle
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
//	30 - Permission error
//	40 - Conflict error
//	45 - Failed precondition error
//	50 - Internal error
const (
	ExitSuccess            = 0
	ExitGeneral            = 1
	ExitUsage              = 2
	ExitInvalid            = 10
	ExitNotFound           = 20
	ExitPermission         = 30
	ExitConflict           = 40
	ExitFailedPrecondition = 45
	ExitInternal           = 50
)

var specs = [...]Spec{
	KindInvalid: {
		Kind:        KindInvalid,
		Name:        "Invalid",
		Message:     "invalid",
		HTTPStatus:  http.StatusBadRequest,
		GRPCCode:    codes.InvalidArgument,
		CLIExitCode: ExitInvalid,
		TUIStyle:    TUIStyleError,
	},
	KindNotFound: {
		Kind:        KindNotFound,
		Name:        "NotFound",
		Message:     "not found",
		HTTPStatus:  http.StatusNotFound,
		GRPCCode:    codes.NotFound,
		CLIExitCode: ExitNotFound,
		TUIStyle:    TUIStyleWarning,
	},
	KindPermission: {
		Kind:        KindPermission,
		Name:        "Permission",
		Message:     "permission denied",
		HTTPStatus:  http.StatusForbidden,
		GRPCCode:    codes.PermissionDenied,
		CLIExitCode: ExitPermission,
		TUIStyle:    TUIStyleError,
	},
	KindConflict: {
		Kind:        KindConflict,
		Name:        "Conflict",
		Message:     "conflict",
		HTTPStatus:  http.StatusConflict,
		GRPCCode:    codes.AlreadyExists,
		CLIExitCode: ExitConflict,
		TUIStyle:    TUIStyleWarning,
	},
	KindFailedPrecondition: {
		Kind:        KindFailedPrecondition,
		Name:        "FailedPrecondition",
		Message:     "failed precondition",
		HTTPStatus:  http.StatusPreconditionFailed,
		GRPCCode:    codes.FailedPrecondition,
		CLIExitCode: ExitFailedPrecondition,
		TUIStyle:    TUIStyleWarning,
	},
	KindInternal: {
		Kind:        KindInternal,
		Name:        "Internal",
		Message:     "internal error",
		HTTPStatus:  http.StatusInternalServerError,
		GRPCCode:    codes.Internal,
		CLIExitCode: ExitInternal,
		TUIStyle:    TUIStyleError,
	},
}

var allKinds = []Kind{
	KindInvalid,
	KindNotFound,
	KindPermission,
	KindConflict,
	KindFailedPrecondition,
	KindInternal,
}

func (k Kind) String() string       { return SpecFor(k).Name }
func (k Kind) HTTPStatus() int      { return SpecFor(k).HTTPStatus }
func (k Kind) GRPCCode() codes.Code { return SpecFor(k).GRPCCode }
func (k Kind) CLIExitCode() int     { return SpecFor(k).CLIExitCode }
func (k Kind) TUIStyle() TUIStyle   { return SpecFor(k).TUIStyle }

// SpecFor returns the immutable transport specification for kind. Unknown
// values safely fall back to the internal-error specification.
func SpecFor(kind Kind) Spec {
	if kind >= kindCount {
		return specs[KindInternal]
	}
	return specs[kind]
}

// AllKinds returns every declared error kind in stable order.
func AllKinds() []Kind {
	return slices.Clone(allKinds)
}
