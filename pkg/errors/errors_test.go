package errors_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/urfave/cli/v3"
	"google.golang.org/grpc/codes"
)

func TestKindMappings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind       errors.Kind
		name       string
		message    string
		httpStatus int
		grpcCode   codes.Code
		cliCode    int
		tuiStyle   errors.TUIStyle
		newError   func() error
		isKind     func(error) bool
	}{
		{errors.KindInvalid, "Invalid", "invalid", http.StatusBadRequest, codes.InvalidArgument, errors.ExitInvalid, errors.TUIStyleError, func() error { return errors.Invalidf("detail") }, errors.IsInvalid},
		{errors.KindNotFound, "NotFound", "not found", http.StatusNotFound, codes.NotFound, errors.ExitNotFound, errors.TUIStyleWarning, func() error { return errors.NotFoundf("detail") }, errors.IsNotFound},
		{errors.KindPermission, "Permission", "permission denied", http.StatusForbidden, codes.PermissionDenied, errors.ExitPermission, errors.TUIStyleError, func() error { return errors.Permissionf("detail") }, errors.IsPermission},
		{errors.KindConflict, "Conflict", "conflict", http.StatusConflict, codes.AlreadyExists, errors.ExitConflict, errors.TUIStyleWarning, func() error { return errors.Conflictf("detail") }, errors.IsConflict},
		{errors.KindFailedPrecondition, "FailedPrecondition", "failed precondition", http.StatusPreconditionFailed, codes.FailedPrecondition, errors.ExitFailedPrecondition, errors.TUIStyleWarning, func() error { return errors.FailedPreconditionf("detail") }, errors.IsFailedPrecondition},
		{errors.KindInternal, "Internal", "internal error", http.StatusInternalServerError, codes.Internal, errors.ExitInternal, errors.TUIStyleError, func() error { return errors.Internalf("detail") }, errors.IsInternal},
	}

	testutil.Equals(t, len(errors.AllKinds()), len(tests))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spec := errors.SpecFor(tt.kind)
			testutil.Equals(t, spec.Kind, tt.kind)
			testutil.Equals(t, spec.Name, tt.name)
			testutil.Equals(t, spec.Message, tt.message)
			testutil.Equals(t, spec.HTTPStatus, tt.httpStatus)
			testutil.Equals(t, spec.GRPCCode, tt.grpcCode)
			testutil.Equals(t, spec.CLIExitCode, tt.cliCode)
			testutil.Equals(t, spec.TUIStyle, tt.tuiStyle)

			err := tt.newError()
			testutil.IsTrue(t, tt.isKind(err))

			var appErr *errors.Error
			testutil.ErrorAs(t, err, &appErr)
			testutil.Equals(t, appErr.Kind(), tt.kind)
			testutil.Equals(t, appErr.HTTPStatus(), tt.httpStatus)
			testutil.Equals(t, appErr.GRPCCode(), tt.grpcCode)
			testutil.Equals(t, appErr.CLIExitCode(), tt.cliCode)
			testutil.Equals(t, appErr.TUIStyle(), tt.tuiStyle)
		})
	}
}

func TestSpecForUnknownKindFallsBackToInternal(t *testing.T) {
	t.Parallel()

	got := errors.SpecFor(errors.Kind(255))
	want := errors.SpecFor(errors.KindInternal)
	testutil.Equals(t, got, want)
}

func TestConstructorUnwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("root")
	err := errors.Internalf("boom: %w", cause)

	testutil.ErrorIsInternal(t, err)
	testutil.ErrorIs(t, err, cause)
	testutil.ErrorIs(t, errors.Unwrap(err), cause)
}

func TestUserMessages(t *testing.T) {
	t.Parallel()

	invalid := errors.Invalidf("name is required")
	testutil.Equals(t, invalid.UserMessage(), "name is required")

	internal := errors.Internalf("database password leaked")
	testutil.Equals(t, internal.UserMessage(), "internal error")

	internal = internal.WithUserMessage("Please try again")
	testutil.Equals(t, internal.UserMessage(), "Please try again")
}

func TestSurfaceAdaptersUseSafeMessage(t *testing.T) {
	t.Parallel()

	err := errors.Internalf("database password leaked")

	tuiErr := errors.ToTUIError(err)
	testutil.Equals(t, tuiErr.Message, "internal error")

	cliErr := errors.ToCLIExit(err)
	var exitErr cli.ExitCoder
	testutil.ErrorAs(t, cliErr, &exitErr)
	testutil.Equals(t, exitErr.ExitCode(), errors.ExitInternal)
	testutil.Equals(t, cliErr.Error(), "internal error")
}

func TestWrappedKind(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("outer: %w", errors.Invalidf("name is required"))
	testutil.ErrorIsInvalid(t, err)
}
