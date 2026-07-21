// Package paging defines cursor-based page requests and results.
package paging

// Cursor is an opaque position in an ordered result set. Cursor pages prevent
// offset-style shifting and duplication, but separate requests do not form a
// database snapshot.
type Cursor string

const DefaultLimit = 100

// Request asks for up to Limit items after Cursor.
type Request struct {
	Cursor Cursor `json:"cursor,omitempty"`
	Limit  int    `json:"limit"`
}

// Page is one window of a cursor-paged result set. Next is empty when there
// are no more visible items.
type Page[T any] struct {
	Items []T    `json:"items"`
	Next  Cursor `json:"next,omitempty"`
}

// Count traverses every page returned by list.
func Count[T any](list func(Cursor) (Page[T], error)) (int, error) {
	count := 0
	var cursor Cursor
	for {
		page, err := list(cursor)
		if err != nil {
			return 0, err
		}
		count += len(page.Items)
		if page.Next == "" {
			return count, nil
		}
		cursor = page.Next
	}
}
