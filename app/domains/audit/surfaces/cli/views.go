package cli

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
)

type AuditRow struct {
	ID        string `table:"ID" json:"id"`
	StartedAt string `table:"STARTED_AT" json:"started_at"`
	Action    string `table:"ACTION" json:"action"`
	Resource  string `table:"RESOURCE" json:"resource"`
	Principal string `table:"PRINCIPAL" json:"principal"`
	Success   bool   `table:"SUCCESS" json:"success"`
	Touches   int    `table:"TOUCHES" json:"touches"`
	Error     string `table:"ERROR" json:"error,omitempty"`
}

func ToAuditRow(entry *models.AuditEntry) AuditRow {
	if entry == nil {
		return AuditRow{}
	}
	return AuditRow{
		ID:        entry.ID.String(),
		StartedAt: formatTime(entry.StartedAt),
		Action:    entry.Action,
		Resource:  entry.Resource.String(),
		Principal: entry.Principal.String(),
		Success:   entry.Success,
		Touches:   len(entry.Touches),
		Error:     entry.Error,
	}
}

func ToAuditRows(entries []*models.AuditEntry) []AuditRow {
	rows := make([]AuditRow, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, ToAuditRow(entry))
	}
	return rows
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
