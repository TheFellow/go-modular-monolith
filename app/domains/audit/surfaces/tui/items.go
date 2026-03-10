package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
)

type auditItem struct {
	entry       models.AuditEntry
	description string
	title       string
}

func newAuditItem(entry models.AuditEntry) auditItem {
	title := fmt.Sprintf("%s %s", entry.StartedAt.Format("15:04:05"), entry.Action)
	description := fmt.Sprintf("%s | %s", entry.Principal.String(), entry.Resource.Type)
	return auditItem{entry: entry, title: title, description: description}
}

func (i auditItem) Title() string { return i.title }
func (i auditItem) Description() string {
	return i.description
}
func (i auditItem) FilterValue() string {
	parts := []string{strings.TrimSpace(i.entry.Action), strings.TrimSpace(string(i.entry.Resource.Type))}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
