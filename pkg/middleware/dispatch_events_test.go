package middleware_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/urfave/cli/v3"
)

type failingEventDispatcher struct{ err error }

func (d failingEventDispatcher) Dispatch(*middleware.Context, any) error { return d.err }

func TestDispatchEventsPreservesApplicationErrorsAndSafePresentation(t *testing.T) {
	t.Parallel()
	for _, cause := range []error{
		errors.FailedPreconditionf("drink is used by a menu"),
		errors.Conflictf("inventory revision changed"),
		errors.Invalidf("replacement ratio is invalid"),
		errors.Permissionf("cannot change this order"),
		errors.NotFoundf("ingredient is missing"),
		errors.Internalf("private database diagnostic"),
		errors.New("unclassified database diagnostic"),
	} {
		t.Run(cause.Error(), func(t *testing.T) {
			t.Parallel()
			f := testutil.NewFixture(t)
			ctx := f.OwnerContext()
			chain := middleware.NewChain(middleware.DispatchEvents(failingEventDispatcher{err: cause}))
			err := chain.Execute(ctx, middleware.Operation{Kind: middleware.OperationKindCommand}, func(ctx *middleware.Context) error {
				ctx.AddEvent("mutation")
				return nil
			})
			testutil.ErrorIs(t, err, cause)
			var original *errors.Error
			wantKind, wantMessage := errors.KindInternal, "internal error"
			if errors.As(cause, &original) {
				wantKind, wantMessage = original.Kind(), original.UserMessage()
			}
			var actual *errors.Error
			testutil.Equals(t, errors.As(err, &actual), true)
			testutil.Equals(t, actual.Kind(), wantKind)
			terminal := errors.ToTUIError(err)
			testutil.Equals(t, terminal.Message, wantMessage)
			testutil.Equals(t, terminal.Style, wantKind.TUIStyle())
			exit := errors.ToCLIExit(err)
			var coder cli.ExitCoder
			testutil.Equals(t, errors.As(exit, &coder), true)
			testutil.Equals(t, coder.ExitCode(), wantKind.CLIExitCode())
			testutil.Equals(t, coder.Error(), wantMessage)
		})
	}
}
