package events

import "github.com/TheFellow/go-modular-monolith/app/kernel/tag"

// TagsReplaced declares the tag state requested by this domain's mutation.
type TagsReplaced struct{ Replacement tag.Replacement }
