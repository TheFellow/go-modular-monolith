package tui

// LabelOr returns fallback when value is empty.
func LabelOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
