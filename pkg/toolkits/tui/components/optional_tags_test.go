//nolint:paralleltest // terminal component focus and input lifecycles run serially.
package components

import (
	"errors"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func TestOptionalTagsDistinguishesPreserveClearAndReplace(t *testing.T) {
	field := NewOptionalTagsField("old")
	desired, err := DesiredTags(field, parseTestTags)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, desired != nil, "%v", "untouched tags did not preserve")
	field.Focus()
	for range len("old") {
		_, _ = field.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	desired, err = DesiredTags(field, parseTestTags)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, desired == nil || *desired != "", "cleared tags = %#v", desired)
	_, _ = field.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("new=value")})
	desired, err = DesiredTags(field, parseTestTags)
	testutil.Ok(t, err)
	testutil.Equals(t, *desired, "new=value")
}

func TestOptionalTagsRejectsInvalidEditedSet(t *testing.T) {
	field := NewOptionalTagsField("")
	field.Focus()
	_, _ = field.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("region=west,region=east")})
	{
		_, err := DesiredTags(field, func(string) (string, error) { return "", errors.New("duplicate") })
		testutil.ErrorIf(t, err == nil, "%v", "duplicate complete tag set accepted")
	}
}
