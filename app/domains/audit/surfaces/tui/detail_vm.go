package tui

import (
	"fmt"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/surfaces"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
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
		d.styles.Muted.Render("Duration: " + surfaces.Duration(entry.StartedAt, entry.CompletedAt)),
		d.styles.Subtitle.Render("Success: ") + fmt.Sprintf("%t", entry.Success),
	}

	if strings.TrimSpace(entry.Error) != "" {
		lines = append(lines, "", d.styles.Subtitle.Render("Error"), entry.Error)
	}

	lines = append(lines,
		"", d.styles.Subtitle.Render("Touched entities"), surfaces.Entities(entry.Touches),
		"", d.styles.Subtitle.Render("Referenced entities"), surfaces.Entities(entry.Participants),
		"", d.styles.Subtitle.Render("Effects"), surfaces.Effects(entry),
	)
	content := strings.Join(lines, "\n")
	if d.width > 0 {
		content = lipgloss.NewStyle().Width(d.width).Render(content)
	}
	return content
}
