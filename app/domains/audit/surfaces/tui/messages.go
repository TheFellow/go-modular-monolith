package tui

import "github.com/TheFellow/go-modular-monolith/app/domains/audit/models"

// AuditLoadedMsg is sent when audit entries have been loaded.
type AuditLoadedMsg struct {
	Entries []models.AuditEntry
	Err     error
}
