package components

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
)

// NewOptionalTagsField creates a complete-tag-set field. An untouched field
// means preserve; editing it to empty means clear; any other value replaces.
func NewOptionalTagsField(current string) *forms.TextField {
	return forms.NewTextField("Complete tags (optional)",
		forms.WithPlaceholder("unchanged; edit to clear or replace"),
		forms.WithInitialValue(current),
	)
}

// DesiredTags translates field interaction into nil/preserve versus
// non-nil/replace semantics using the caller's parser.
func DesiredTags[T any](field *forms.TextField, parse func(string) (T, error)) (*T, error) {
	if field == nil || !field.IsDirty() {
		return nil, nil
	}
	raw, _ := field.Value().(string)
	values, err := parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, errors.Invalidf("invalid tags: %v", err)
	}
	return &values, nil
}
