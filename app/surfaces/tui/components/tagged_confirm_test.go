//nolint:paralleltest // terminal component focus and input lifecycles run serially.
package components

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/dialog"
	tea "github.com/charmbracelet/bubbletea"
)

func TestTaggedConfirmAdvancesFromTagChoiceToDomainConfirmation(t *testing.T) {
	component := NewTaggedConfirm(tag.Tags{{Key: "region", Value: "west"}}, dialog.NewConfirmDialog("Complete order", "Complete it?"))
	testutil.StringContains(t, component.View(), "Complete tags (optional)")

	component, _ = component.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	testutil.StringContains(t, component.View(), "Complete order")
	testutil.StringContains(t, component.View(), "Complete it?")

	desired, err := component.DesiredTags()
	testutil.Ok(t, err)
	if desired != nil {
		testutil.ErrorIf(t, true, "%v", "untouched transition tags must preserve the existing set")
	}
}
