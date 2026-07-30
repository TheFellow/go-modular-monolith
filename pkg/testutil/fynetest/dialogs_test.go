//nolint:paralleltest // Fyne test dialog state is process-global.
package fynetest

import (
	"errors"
	"testing"
)

func TestDialogsRecordConfirmationResponseAndErrors(t *testing.T) {
	dialogs := &Dialogs{}
	responded := false
	dialogs.Confirm("Delete", "Delete the drink?", func(confirmed bool) { responded = confirmed })
	wantErr := errors.New("save failed")
	dialogs.ShowError(wantErr)

	confirmations := dialogs.Confirmations()
	if len(confirmations) != 1 || confirmations[0].Title != "Delete" || confirmations[0].Message != "Delete the drink?" {
		t.Fatalf("confirmations = %#v", confirmations)
	}
	confirmations[0].Respond(true)
	if !responded {
		t.Fatal("recorded confirmation did not retain response")
	}
	if got := dialogs.Errors(); len(got) != 1 || !errors.Is(got[0], wantErr) {
		t.Fatalf("errors = %v", got)
	}
}
