package ids

import (
	"fmt"
	"strings"
	"time"

	"github.com/cedar-policy/cedar-go"
	"github.com/segmentio/ksuid"
)

var prefixes = map[cedar.EntityType]string{
	"Mixology::Drink":      "drk",
	"Mixology::Ingredient": "ing",
	"Mixology::Menu":       "mnu",
	"Mixology::Order":      "ord",
	"Mixology::Inventory":  "inv",
	"Mixology::AuditEntry": "aud",
}

func New(entityType cedar.EntityType) (cedar.EntityUID, error) {
	id := ksuid.New()

	prefix := prefixFor(entityType)
	if prefix == "" {
		prefix = derivePrefix(entityType)
	}

	idStr := fmt.Sprintf("%s-%s", prefix, id.String())
	return cedar.NewEntityUID(entityType, cedar.String(idStr)), nil
}

// Parse extracts the KSUID from a prefixed ID string (e.g. "drk-<ksuid>").
func Parse(idStr string) (ksuid.KSUID, error) {
	parts := strings.SplitN(idStr, "-", 2)
	if len(parts) != 2 {
		return ksuid.Nil, fmt.Errorf("invalid id format: %s", idStr)
	}
	return ksuid.Parse(parts[1])
}

// Time extracts the embedded timestamp from a prefixed KSUID ID string.
func Time(idStr string) (time.Time, error) {
	id, err := Parse(idStr)
	if err != nil {
		return time.Time{}, err
	}
	return id.Time(), nil
}

func prefixFor(entityType cedar.EntityType) string {
	return prefixes[entityType]
}

func derivePrefix(entityType cedar.EntityType) string {
	s := string(entityType)
	if idx := strings.LastIndex(s, "::"); idx >= 0 {
		s = s[idx+2:]
	}
	s = strings.ToLower(s)
	if len(s) > 3 {
		s = s[:3]
	}
	return s
}
