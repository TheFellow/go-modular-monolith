package components

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
)

// NewOptionalTagsField creates a complete-tag-set field. An untouched field
// means preserve; editing it to empty means clear; any other value replaces.
func NewOptionalTagsField(current tag.Tags) *forms.TextField {
	return forms.NewTextField("Complete tags (optional)",
		forms.WithPlaceholder("unchanged; edit to clear or replace"),
		forms.WithInitialValue(current.Canonical().String()),
	)
}

// DesiredTags translates the field's interaction state into the application
// service's nil/preserve versus non-nil/replace contract.
func DesiredTags(field *forms.TextField) (*tag.Tags, error) {
	if field == nil || !field.IsDirty() {
		return nil, nil
	}
	raw, _ := field.Value().(string)
	values, err := tag.ParseCollection(strings.TrimSpace(raw))
	if err != nil {
		return nil, errors.Invalidf("invalid tags: %v", err)
	}
	return &values, nil
}
