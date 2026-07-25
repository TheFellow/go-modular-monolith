// Package tag defines user-authored metadata shared by every application domain.
package tag

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

const (
	// MaxKeyLength bounds tag keys in Unicode code points.
	MaxKeyLength = 64
	// MaxValueLength bounds tag values in Unicode code points.
	MaxValueLength = 256
)

// Tag is user-authored metadata identified by Key. Value is optional.
// Keys are unique within the Tags attached to an entity.
type Tag struct {
	Key   string
	Value string
}

// New validates and returns a Tag after trimming outer whitespace.
func New(key, value string) (Tag, error) {
	t := Tag{
		Key:   strings.TrimSpace(key),
		Value: strings.TrimSpace(value),
	}
	if err := t.Validate(); err != nil {
		return Tag{}, err
	}
	return t, nil
}

// Parse accepts the canonical forms "key" and "key=value". Only the first
// equals sign separates the key and value.
func Parse(value string) (Tag, error) {
	key, tagValue, _ := strings.Cut(value, "=")
	return New(key, tagValue)
}

// Validate reports whether t can be used as an application tag.
func (t Tag) Validate() error {
	if !utf8.ValidString(t.Key) || !utf8.ValidString(t.Value) {
		return errors.Invalidf("tag must contain valid UTF-8")
	}
	if strings.TrimSpace(t.Key) == "" {
		return errors.Invalidf("tag key is required")
	}
	if t.Key != strings.TrimSpace(t.Key) {
		return errors.Invalidf("tag key must not have outer whitespace")
	}
	if strings.Contains(t.Key, "=") {
		return errors.Invalidf("tag key must not contain =")
	}
	if strings.ContainsFunc(t.Key, unicode.IsControl) || strings.ContainsFunc(t.Value, unicode.IsControl) {
		return errors.Invalidf("tag must not contain control characters")
	}
	if t.Value != strings.TrimSpace(t.Value) {
		return errors.Invalidf("tag value must not have outer whitespace")
	}
	if utf8.RuneCountInString(t.Key) > MaxKeyLength {
		return errors.Invalidf("tag key must be at most %d characters", MaxKeyLength)
	}
	if utf8.RuneCountInString(t.Value) > MaxValueLength {
		return errors.Invalidf("tag value must be at most %d characters", MaxValueLength)
	}
	return nil
}

// String returns the canonical user-facing spelling of t.
func (t Tag) String() string {
	if t.Value == "" {
		return t.Key
	}
	return fmt.Sprintf("%s=%s", t.Key, t.Value)
}

// Tags is an entity's collection of tags. Its helpers return deterministic,
// key-sorted copies and do not mutate their receiver.
type Tags []Tag

// Validate checks every tag and rejects duplicate keys.
func (tags Tags) Validate() error {
	seen := make(map[string]struct{}, len(tags))
	for _, t := range tags {
		if err := t.Validate(); err != nil {
			return err
		}
		if _, ok := seen[t.Key]; ok {
			return errors.Invalidf("duplicate tag key: %s", t.Key)
		}
		seen[t.Key] = struct{}{}
	}
	return nil
}

// Sorted returns a key-sorted copy of tags.
func (tags Tags) Sorted() Tags {
	out := slices.Clone(tags)
	slices.SortFunc(out, func(a, b Tag) int {
		return strings.Compare(a.Key, b.Key)
	})
	return out
}

// Upsert adds next or replaces the tag with the same key.
func (tags Tags) Upsert(next Tag) Tags {
	out := make(Tags, 0, len(tags)+1)
	for _, t := range tags {
		if t.Key != next.Key {
			out = append(out, t)
		}
	}
	out = append(out, next)
	return out.Sorted()
}

// Remove removes the tag identified by key. A missing key is a no-op.
func (tags Tags) Remove(key string) Tags {
	key = strings.TrimSpace(key)
	out := make(Tags, 0, len(tags))
	for _, t := range tags {
		if t.Key != key {
			out = append(out, t)
		}
	}
	return out.Sorted()
}

// Strings returns canonical user-facing spellings sorted by key.
func (tags Tags) Strings() []string {
	sorted := tags.Sorted()
	out := make([]string, 0, len(sorted))
	for _, t := range sorted {
		out = append(out, t.String())
	}
	return out
}

// Map returns a copy keyed by tag key. Duplicate keys use the last value.
func (tags Tags) Map() map[string]string {
	out := make(map[string]string, len(tags))
	for _, t := range tags {
		out[t.Key] = t.Value
	}
	return out
}

// FromMap validates and returns tags sorted by key.
func FromMap(values map[string]string) (Tags, error) {
	tags := make(Tags, 0, len(values))
	for key, value := range values {
		t, err := New(key, value)
		if err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	if err := tags.Validate(); err != nil {
		return nil, err
	}
	return tags.Sorted(), nil
}
