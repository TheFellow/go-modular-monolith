package tui

import "github.com/TheFellow/go-modular-monolith/app/domains/audit/models"

import "github.com/TheFellow/go-modular-monolith/pkg/paging"

// AuditLoadedMsg is sent when audit entries have been loaded.
type AuditLoadedMsg struct {
	Entries []models.AuditEntry
	Err     error
	Next    paging.Cursor
	Token   uint64
}
