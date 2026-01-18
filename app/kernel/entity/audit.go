package entity

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

const (
	TypeAuditEntry   = cedar.EntityType("Mixology::AuditEntry")
	PrefixAuditEntry = "aud"
)

// AuditEntryID is a strongly typed ID for audit entry entities.
type AuditEntryID cedar.EntityUID

// NewAuditEntryID generates a new audit entry ID with a KSUID.
func NewAuditEntryID() AuditEntryID {
	return AuditEntryID(NewID(TypeAuditEntry, PrefixAuditEntry))
}

// ParseAuditEntryID creates an audit entry ID from a stored string.
func ParseAuditEntryID(id string) (AuditEntryID, error) {
	if id == "" {
		return AuditEntryID(cedar.NewEntityUID(TypeAuditEntry, cedar.String(""))), nil
	}
	if !strings.HasPrefix(id, PrefixAuditEntry+"-") {
		return AuditEntryID{}, errors.Invalidf("invalid audit entry id prefix: %s", id)
	}
	return AuditEntryID(cedar.NewEntityUID(TypeAuditEntry, cedar.String(id))), nil
}

// EntityUID converts the ID to a cedar.EntityUID.
func (id AuditEntryID) EntityUID() cedar.EntityUID {
	return cedar.EntityUID(id)
}

// String returns the string form of the ID.
func (id AuditEntryID) String() string {
	return string(cedar.EntityUID(id).ID)
}

// IsZero returns true when the ID is unset.
func (id AuditEntryID) IsZero() bool {
	return cedar.EntityUID(id).ID == ""
}
