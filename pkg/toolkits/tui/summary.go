package tui

import "strings"

// ListSummary joins compact list metadata while omitting unavailable values.
func ListSummary(parts ...string) string {
	visible := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			visible = append(visible, value)
		}
	}
	return strings.Join(visible, " • ")
}

// TagSummary is the terminal equivalent of grid tag pills. The list delegate
// handles viewport truncation while retaining the canonical leading tags.
func TagSummary(canonical string) string {
	canonical = strings.TrimSpace(canonical)
	if canonical == "" {
		return ""
	}
	return "tags: " + canonical
}
