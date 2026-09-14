package cli

import (
	"time"

	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	cedar "github.com/cedar-policy/cedar-go"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/surfaces"
)

type AuditRow struct {
	Effects      []middlewareevents.Effect `table:"-" json:"effects"`
	Participants []cedar.EntityUID         `table:"-" json:"participants"`
	ID           string                    `table:"ID" json:"id"`
	StartedAt    string                    `table:"STARTED_AT" json:"started_at"`
	CompletedAt  string                    `table:"COMPLETED_AT" json:"completed_at"`
	Duration     string                    `table:"DURATION" json:"duration"`
	Action       string                    `table:"ACTION" json:"action"`
	Resource     string                    `table:"RESOURCE" json:"resource"`
	Principal    string                    `table:"PRINCIPAL" json:"principal"`
	Success      bool                      `table:"SUCCESS" json:"success"`
	Touches      int                       `table:"TOUCHES" json:"touches"`
	Error        string                    `table:"ERROR" json:"error,omitempty"`
}

func ToAuditRow(entry *models.AuditEntry) AuditRow {
	if entry == nil {
		return AuditRow{}
	}
	return AuditRow{Effects: entry.Effects, Participants: entry.Participants,
		ID:          entry.ID.String(),
		StartedAt:   formatTime(entry.StartedAt),
		CompletedAt: formatTime(entry.CompletedAt),
		Duration:    surfaces.Duration(entry.StartedAt, entry.CompletedAt),
		Action:      entry.Action,
		Resource:    entry.Resource.String(),
		Principal:   entry.Principal.String(),
		Success:     entry.Success,
		Touches:     len(entry.Touches),
		Error:       entry.Error,
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
