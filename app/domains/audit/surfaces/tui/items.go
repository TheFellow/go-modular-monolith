package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/mvvm"
)

type auditItem = mvvm.ListItem[models.AuditEntry]

func newAuditItem(entry models.AuditEntry) auditItem {
	title := fmt.Sprintf("%s %s", entry.StartedAt.Format("15:04:05"), entry.Action)
	description := fmt.Sprintf("%s | %s", entry.Principal.String(), entry.Resource.Type)
	parts := []string{strings.TrimSpace(entry.Action), strings.TrimSpace(string(entry.Resource.Type))}
	filterValue := strings.TrimSpace(strings.Join(parts, " "))
	return mvvm.NewListItem(entry, title, description, filterValue)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
