// Package paging defines cursor-based page requests and results.
package paging

// Cursor is an opaque position in an ordered result set.
type Cursor string

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
