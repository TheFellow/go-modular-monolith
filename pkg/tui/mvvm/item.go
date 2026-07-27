// Package mvvm contains small, presentation-only contracts shared by TUI view
// models. Domain models remain concrete; these types describe only the stable
// shapes consumed by standard Bubble Tea components.
package mvvm

// ListItem adapts a domain value to the presentation contract expected by a
// bubbles/list.Model. Keeping the value typed lets a domain ViewModel recover
// its selection without weakening its application-layer model.
type ListItem[T any] struct {
	Value       T
	title       string
	description string
	filterValue string
}

// NewListItem creates a typed presentation item. An empty filter value falls
// back to the title, which is the least surprising behavior for simple lists.
func NewListItem[T any](value T, title, description, filterValue string) ListItem[T] {
	if filterValue == "" {
		filterValue = title
	}
	return ListItem[T]{
		Value:       value,
		title:       title,
		description: description,
		filterValue: filterValue,
	}
}

func (i ListItem[T]) Title() string       { return i.title }
func (i ListItem[T]) Description() string { return i.description }
func (i ListItem[T]) FilterValue() string { return i.filterValue }
