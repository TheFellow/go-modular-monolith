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

	if got := len(errors.AllKinds()); got != len(tests) {
		t.Fatalf("AllKinds() returned %d kinds, want %d", got, len(tests))
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spec := errors.SpecFor(tt.kind)
			if spec.Kind != tt.kind || spec.Name != tt.name || spec.Message != tt.message ||
				spec.HTTPStatus != tt.httpStatus || spec.GRPCCode != tt.grpcCode ||
				spec.CLIExitCode != tt.cliCode || spec.TUIStyle != tt.tuiStyle {
				t.Fatalf("SpecFor(%v) = %+v", tt.kind, spec)
			}

			err := tt.newError()
			if !tt.isKind(err) {
				t.Fatalf("kind checker rejected %T", err)
			}

			var appErr *errors.Error
			testutil.ErrorAs(t, err, &appErr)
			if appErr.Kind() != tt.kind || appErr.HTTPStatus() != tt.httpStatus ||
				appErr.GRPCCode() != tt.grpcCode || appErr.CLIExitCode() != tt.cliCode ||
				appErr.TUIStyle() != tt.tuiStyle {
				t.Fatalf("shared error has unexpected mappings: kind=%v HTTP=%d gRPC=%v CLI=%d TUI=%v",
					appErr.Kind(), appErr.HTTPStatus(), appErr.GRPCCode(), appErr.CLIExitCode(), appErr.TUIStyle())
			}
		})
	}
}

func TestSpecForUnknownKindFallsBackToInternal(t *testing.T) {
	t.Parallel()

	got := errors.SpecFor(errors.Kind(255))
	want := errors.SpecFor(errors.KindInternal)
	if got != want {
		t.Fatalf("SpecFor(unknown) = %+v, want %+v", got, want)
	}
}

func TestConstructorUnwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("root")
	err := errors.Internalf("boom: %w", cause)

	testutil.ErrorIsInternal(t, err)
	testutil.ErrorIs(t, err, cause)
	if !errors.Is(errors.Unwrap(err), cause) {
		t.Fatalf("Unwrap() = %v, want %v", errors.Unwrap(err), cause)
	}
}

func TestUserMessages(t *testing.T) {
	t.Parallel()

	invalid := errors.Invalidf("name is required")
	if got := invalid.UserMessage(); got != "name is required" {
		t.Fatalf("invalid UserMessage() = %q", got)
	}

	internal := errors.Internalf("database password leaked")
	if got := internal.UserMessage(); got != "internal error" {
		t.Fatalf("internal UserMessage() = %q", got)
	}

	internal = internal.WithUserMessage("Please try again")
	if got := internal.UserMessage(); got != "Please try again" {
		t.Fatalf("overridden UserMessage() = %q", got)
	}
}

func TestSurfaceAdaptersUseSafeMessage(t *testing.T) {
	t.Parallel()

	err := errors.Internalf("database password leaked")

	tuiErr := errors.ToTUIError(err)
	if tuiErr.Message != "internal error" {
		t.Fatalf("TUI message = %q", tuiErr.Message)
	}

	cliErr := errors.ToCLIExit(err)
	var exitErr cli.ExitCoder
	ok := errors.As(cliErr, &exitErr)
	if !ok {
		t.Fatalf("ToCLIExit() returned %T", cliErr)
	}
	if exitErr.ExitCode() != errors.ExitInternal || cliErr.Error() != "internal error" {
		t.Fatalf("CLI error = %q code=%d", cliErr.Error(), exitErr.ExitCode())
	}
}

func TestWrappedKind(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("outer: %w", errors.Invalidf("name is required"))
	if !errors.IsInvalid(err) {
		t.Fatalf("IsInvalid(%v) = false", err)
	}
}
