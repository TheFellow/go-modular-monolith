package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/cedar-policy/cedar-go"
	"github.com/charmbracelet/lipgloss"
)

// DetailViewModel renders an audit detail pane.
type DetailViewModel struct {
	styles tui.ListViewStyles
	width  int
	height int
	entry  optional.Value[models.AuditEntry]
}

func NewDetailViewModel(styles tui.ListViewStyles) *DetailViewModel {
	return &DetailViewModel{styles: styles}
}

func (d *DetailViewModel) SetSize(width, height int) {
	d.width = width
	d.height = height
}

func (d *DetailViewModel) SetEntry(entry optional.Value[models.AuditEntry]) {
	d.entry = entry
}

func (d *DetailViewModel) View() string {
	entry, ok := d.entry.Unwrap()
	if !ok {
		return d.styles.Subtitle.Render("Select an entry to view details")
	}

	lines := []string{
		d.styles.Title.Render("Audit Entry"),
		d.styles.Muted.Render("ID: " + entry.ID.String()),
		d.styles.Subtitle.Render("Action: ") + entry.Action,
		d.styles.Subtitle.Render("Principal: ") + entry.Principal.String(),
		d.styles.Subtitle.Render("Resource: ") + entry.Resource.String(),
		d.styles.Muted.Render("Started: " + formatTime(entry.StartedAt)),
		d.styles.Muted.Render("Completed: " + formatTime(entry.CompletedAt)),
		d.styles.Subtitle.Render("Success: ") + fmt.Sprintf("%t", entry.Success),
	}

	if strings.TrimSpace(entry.Error) != "" {
		lines = append(lines, "", d.styles.Subtitle.Render("Error"), entry.Error)
	}

	touched := touchedEntities(entry.Touches)
	lines = append(lines, "", d.styles.Subtitle.Render("Touched Entities"))
	lines = append(lines, touched...)

	content := strings.Join(lines, "\n")
	if d.width > 0 {
		content = lipgloss.NewStyle().Width(d.width).Render(content)
	}
	return content
}

func touchedEntities(entities []cedar.EntityUID) []string {
	if len(entities) == 0 {
		return []string{"(none)"}
	}

	sorted := make([]string, 0, len(entities))
	for _, uid := range entities {
		sorted = append(sorted, uid.String())
	}
	sort.Strings(sorted)

	lines := make([]string, 0, len(sorted))
	for _, uid := range sorted {
		lines = append(lines, "- "+uid)
	}
	return lines
}
