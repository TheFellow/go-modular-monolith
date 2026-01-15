package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	cedar "github.com/cedar-policy/cedar-go"
)

func toRow(e models.AuditEntry) AuditEntryRow {
	return AuditEntryRow{
		ID:          string(e.ID.ID),
		Action:      e.Action,
		ResourceType: string(e.Resource.Type),
		ResourceID:   string(e.Resource.ID),
		PrincipalType: string(e.Principal.Type),
		PrincipalID:   string(e.Principal.ID),
		Touches:     e.Touches,
		StartedAt:   e.StartedAt,
		CompletedAt: e.CompletedAt,
		Success:     e.Success,
		Error:       e.Error,
	}
}

func toModel(r AuditEntryRow) models.AuditEntry {
	return models.AuditEntry{
		ID:          cedar.NewEntityUID(models.AuditEntryEntityType, cedar.String(r.ID)),
		Action:      r.Action,
		Resource:    cedar.NewEntityUID(cedar.EntityType(r.ResourceType), cedar.String(r.ResourceID)),
		Principal:   cedar.NewEntityUID(cedar.EntityType(r.PrincipalType), cedar.String(r.PrincipalID)),
		Touches:     r.Touches,
		StartedAt:   r.StartedAt,
		CompletedAt: r.CompletedAt,
		Success:     r.Success,
		Error:       r.Error,
	}
}
