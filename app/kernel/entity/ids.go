package entity

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/segmentio/ksuid"
)

// NewID generates a KSUID-based ID with the given prefix.
func NewID(entityType cedar.EntityType, prefix string) cedar.EntityUID {
	return cedar.NewEntityUID(entityType, cedar.String(prefix+"-"+ksuid.New().String()))
}

func parseID(entityType cedar.EntityType, prefix, id string) (cedar.EntityUID, error) {
	if id == "" {
		return cedar.EntityUID{}, errors.Invalidf("invalid %s id: empty", prefix)
	}

	suffix, ok := strings.CutPrefix(id, prefix+"-")
	if !ok {
		return cedar.EntityUID{}, errors.Invalidf("invalid %s id prefix: %s", prefix, id)
	}
	if _, err := ksuid.Parse(suffix); err != nil {
		return cedar.EntityUID{}, errors.Invalidf("invalid %s id: %s", prefix, id)
	}
	return cedar.NewEntityUID(entityType, cedar.String(id)), nil
}
