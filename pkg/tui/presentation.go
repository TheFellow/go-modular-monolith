package tui

// TagLabel returns the standard TUI label for a canonical tag string.
func TagLabel(value string) string {
	if value == "" {
		return "(none)"
	}
	return value
}
