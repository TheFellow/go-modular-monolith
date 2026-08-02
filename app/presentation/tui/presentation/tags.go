// Package presentation contains Mixology-specific TUI presentation policy.
package presentation

// TagLabel returns the standard display value for a canonical tag string.
func TagLabel(value string) string {
	if value == "" {
		return "(none)"
	}
	return value
}
