// Package surfaces provides shared, framework-independent audit presentation.
package surfaces

import (
	"sort"
	"strings"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	cedar "github.com/cedar-policy/cedar-go"
)

// Duration uses the same elapsed time precision on every audit surface.
func Duration(start, completed time.Time) string {
	if start.IsZero() || completed.IsZero() || completed.Before(start) {
		return ""
	}
	return completed.Sub(start).Round(time.Microsecond).String()
}

// Workflow describes the correlation ID, including its absence in older entries.
func Workflow(entry models.AuditEntry) string {
	if entry.WorkflowID == "" {
		return "(none)"
	}
	return entry.WorkflowID
}

// Entities gives references a stable, readable order without changing the entry.
func Entities(entities []cedar.EntityUID) string {
	if len(entities) == 0 {
		return "(none)"
	}
	lines := make([]string, 0, len(entities))
	for _, uid := range entities {
		lines = append(lines, "- "+uid.String())
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// Effects distinguishes recorded changes from attempts rolled back on failure.
func Effects(entry models.AuditEntry) string {
	if len(entry.Effects) == 0 {
		return "(none)"
	}
	outcome := "Committed effects"
	if !entry.Success {
		outcome = "Attempted effects (not committed)"
	}
	lines := []string{outcome}
	for _, effect := range entry.Effects {
		lines = append(lines, effect.Kind+": "+effect.Resource.String())
		for _, change := range effect.Changes {
			lines = append(lines, "  "+change.Field+":", "    Before: "+change.Before, "    After: "+change.After)
		}
	}
	return strings.Join(lines, "\n")
}
