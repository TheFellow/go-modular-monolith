package entity

import (
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/segmentio/ksuid"
)

// NewID generates a KSUID-based ID with the given prefix.
func NewID(entityType cedar.EntityType, prefix string) cedar.EntityUID {
	return cedar.NewEntityUID(entityType, cedar.String(prefix+"-"+ksuid.New().String()))
}
