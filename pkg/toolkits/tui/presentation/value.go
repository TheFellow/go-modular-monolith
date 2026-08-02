// Package presentation contains small, domain-neutral display helpers.
package presentation

// LabelOr returns fallback when value is empty.
func LabelOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
