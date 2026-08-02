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
	testutil.ErrorIf(t, desired != nil, "%v", "untouched tags did not preserve")
	field.Focus()
	for range len("old") {
		_, _ = field.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	desired, err = DesiredTags(field)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, desired == nil || len(*desired) != 0, "cleared tags = %#v", desired)
	_, _ = field.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("new=value")})
	desired, err = DesiredTags(field)
	testutil.Ok(t, err)
	testutil.Equals(t, desired.Canonical().String(), "new=value")
}

func TestOptionalTagsRejectsInvalidEditedSet(t *testing.T) {
	field := NewOptionalTagsField(nil)
	field.Focus()
	_, _ = field.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("region=west,region=east")})
	{
		_, err := DesiredTags(field)
		testutil.ErrorIf(t, err == nil, "%v", "duplicate complete tag set accepted")
	}
}
