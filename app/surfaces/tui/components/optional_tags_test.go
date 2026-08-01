//nolint:paralleltest // terminal component focus and input lifecycles run serially.
package components

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func TestOptionalTagsDistinguishesPreserveClearAndReplace(t *testing.T) {
	field := NewOptionalTagsField(tag.Tags{{Key: "old"}})
	desired, err := DesiredTags(field)
	testutil.Ok(t, err)
	if desired != nil {
		testutil.ErrorIf(t, true, "%v", "untouched tags did not preserve")
	}
	field.Focus()
	for range len("old") {
		_, _ = field.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	desired, err = DesiredTags(field)
	testutil.Ok(t, err)
	if desired == nil || len(*desired) != 0 {
		testutil.ErrorIf(t, true, "cleared tags = %#v", desired)
	}
	_, _ = field.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("new=value")})
	desired, err = DesiredTags(field)
	testutil.Ok(t, err)
	testutil.Equals(t, desired.Canonical().String(), "new=value")
}

func TestOptionalTagsRejectsInvalidEditedSet(t *testing.T) {
	field := NewOptionalTagsField(nil)
	field.Focus()
	_, _ = field.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("region=west,region=east")})
	if _, err := DesiredTags(field); err == nil {
		testutil.ErrorIf(t, true, "%v", "duplicate complete tag set accepted")
	}
}
