package tag

import (
	"slices"

	cedar "github.com/cedar-policy/cedar-go"
)

// Edit is an optional complete replacement supplied to an owning domain command.
// A nil Desired preserves tags; an empty non-nil set clears them. Expected
// captures the complete set read by an editor for optimistic concurrency.
type Edit struct {
	Desired  *Tags
	Expected *Tags
}

func Replace(desired *Tags, expected ...Tags) Edit {
	edit := Edit{}
	if desired != nil {
		values := slices.Clone(*desired)
		edit.Desired = &values
	}
	if len(expected) > 0 {
		values := slices.Clone(expected[0])
		edit.Expected = &values
	}
	return edit
}

// Replacement is carried by a consuming domain's event. The owner supplies its
// authorization actions; Tagging owns validation and association persistence.
type Replacement struct {
	Edit        Edit
	Entity      cedar.Entity
	TagAction   cedar.EntityUID
	UntagAction cedar.EntityUID
}
