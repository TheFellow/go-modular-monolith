package tui

// ListItem adapts a typed value to bubbles/list.Item.
type ListItem[T any] struct {
	Value       T
	title       string
	description string
	filterValue string
}

// NewListItem creates a typed list item. An empty filter value uses the title.
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
