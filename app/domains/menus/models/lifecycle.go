package models

import "github.com/TheFellow/go-modular-monolith/pkg/errors"

// RequireDraft ensures a mutation that is only valid while editing a menu can
// proceed. Keeping this rule with the model lets command handlers and UI
// surfaces evaluate the same business precondition.
func (m Menu) RequireDraft() error {
	if m.Status == MenuStatusDraft {
		return nil
	}

	return errors.FailedPreconditionf("menu %q must be draft, got %q", m.ID.String(), m.Status)
}

// RequirePublishable ensures the menu is in a state that may be published.
func (m Menu) RequirePublishable() error {
	if err := m.RequireDraft(); err != nil {
		return err
	}
	if len(m.Items) == 0 {
		return errors.FailedPreconditionf("menu %q must contain at least one item to be published", m.ID.String())
	}

	return nil
}

// RequireReturnToDraft ensures the menu is currently published before it is
// returned to draft.
func (m Menu) RequireReturnToDraft() error {
	if m.Status == MenuStatusPublished {
		return nil
	}

	return errors.FailedPreconditionf("menu %q must be published, got %q", m.ID.String(), m.Status)
}
