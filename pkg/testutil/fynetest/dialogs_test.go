//nolint:paralleltest // Fyne test dialog state is process-global.
package fynetest

import (
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"testing"
)

func TestDialogsRecordConfirmationResponseAndErrors(t *testing.T) {
	dialogs := &Dialogs{}
	responded := false
	dialogs.Confirm("Delete", "Delete the drink?", func(confirmed bool) { responded = confirmed })
	wantErr := errors.New("save failed")
	dialogs.ShowError(wantErr)

	confirmations := dialogs.Confirmations()
	testutil.ErrorIf(t, len(confirmations) != 1 || confirmations[0].Title != "Delete" || confirmations[0].Message != "Delete the drink?", "confirmations = %#v", confirmations)
	confirmations[0].Respond(true)
	testutil.ErrorIf(t, !responded, "%v", "recorded confirmation did not retain response")
	{
		got := dialogs.Errors()
		testutil.ErrorIf(t, len(got) != 1 || !errors.Is(got[0], wantErr), "errors = %v", got)
	}
}
