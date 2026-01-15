package entity

import "github.com/cedar-policy/cedar-go"

const (
	TypeAuditEntry   = cedar.EntityType("Mixology::AuditEntry")
	PrefixAuditEntry = "aud"
)

func AuditEntryID(id string) cedar.EntityUID {
	return cedar.NewEntityUID(TypeAuditEntry, cedar.String(id))
}

func NewAuditEntryID() cedar.EntityUID {
	return NewID(TypeAuditEntry, PrefixAuditEntry)
}
