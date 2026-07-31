package gui_test

import (
	"errors"
	"testing"

	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
	gui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

func TestPresentErrorClassifiesKindsAndProtectsInternalDetail(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		err      error
		severity gui.ErrorSeverity
		message  string
	}{
		{"invalid", apperrors.Invalidf("name is required"), gui.ErrorSeverityInline, "name is required"},
		{"not found", apperrors.NotFoundf("drink missing"), gui.ErrorSeverityWarning, "drink missing"},
		{"permission", apperrors.Permissionf("may not edit"), gui.ErrorSeverityError, "may not edit"},
		{"conflict", apperrors.Conflictf("name already exists"), gui.ErrorSeverityWarning, "name already exists"},
		{"precondition", apperrors.FailedPreconditionf("menu is published"), gui.ErrorSeverityWarning, "menu is published"},
		{"internal", apperrors.Internalf("database password leaked"), gui.ErrorSeverityError, "internal error"},
		{"unknown", errors.New("driver secret"), gui.ErrorSeverityError, "internal error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := gui.PresentError(test.err)
			var presentation gui.ErrorPresentation
			if !errors.As(got, &presentation) {
				t.Fatalf("presentation type = %T", got)
			}
			if presentation.Severity != test.severity || got.Error() != test.message {
				t.Fatalf("presentation = %#v", presentation)
			}
			if !errors.Is(got, test.err) {
				t.Fatal("presentation did not retain its cause")
			}
		})
	}
}

func TestPresentErrorHandlesNilWrappedAndExplicitSafeMessages(t *testing.T) {
	t.Parallel()
	if got := gui.PresentError(nil); got != nil {
		t.Fatalf("nil presentation = %#v", got)
	}

	cause := apperrors.Conflictf("diagnostic detail").WithUserMessage("That name is already in use")
	wrapped := errors.Join(errors.New("operation failed"), cause)
	got := gui.PresentError(wrapped)
	var presentation gui.ErrorPresentation
	if !errors.As(got, &presentation) {
		t.Fatalf("presentation type = %T", got)
	}
	if presentation.Severity != gui.ErrorSeverityWarning || presentation.Message != "That name is already in use" {
		t.Fatalf("presentation = %#v", presentation)
	}
	if !errors.Is(got, cause) {
		t.Fatal("presentation did not retain wrapped typed cause")
	}
}

func TestShowPresentationUsesSeverityAndSuppressesInlineValidation(t *testing.T) {
	t.Parallel()
	dialogs := &fynetest.Dialogs{}
	gui.ShowPresentation(dialogs, apperrors.Invalidf("fix field"))
	gui.ShowPresentation(dialogs, apperrors.Conflictf("already exists"))
	permission := apperrors.Permissionf("denied")
	gui.ShowPresentation(dialogs, permission)
	if len(dialogs.Warnings()) != 1 || dialogs.Warnings()[0].Message != "already exists" {
		t.Fatalf("warnings = %#v", dialogs.Warnings())
	}
	if len(dialogs.Errors()) != 1 || dialogs.Errors()[0].Error() != "denied" {
		t.Fatalf("errors = %#v", dialogs.Errors())
	}
	if !errors.Is(dialogs.Errors()[0], permission) {
		t.Fatal("error dialog discarded typed cause")
	}
}
