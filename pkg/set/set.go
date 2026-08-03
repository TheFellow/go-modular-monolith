// Package set provides a generic unordered set of comparable elements.
package set

import (
	"iter"
	"maps"
)

// Set is an unordered collection of unique elements. The zero value is an
// empty set ready for use.
//
// Like a map, assigning a Set copies a reference to the same elements. Use
// Clone when an independent set is required.
type Set[T comparable] struct {
	m map[T]struct{}
}

// New returns a Set containing items.
func New[T comparable](items ...T) Set[T] {
	var s Set[T]
	s.Add(items...)
	return s
}

// Collect returns a Set containing the elements of seq.
func Collect[T comparable](seq iter.Seq[T]) Set[T] {
	var s Set[T]
	for item := range seq {
		s.Add(item)
	}
	return s
}

// Add inserts items into the set.
func (s *Set[T]) Add(items ...T) {
	if s.m == nil && len(items) > 0 {
		s.m = make(map[T]struct{}, len(items))
	}
	for _, item := range items {
		s.m[item] = struct{}{}
	}
}

// Remove deletes items from the set. Missing items are ignored.
func (s *Set[T]) Remove(items ...T) {
	for _, item := range items {
		delete(s.m, item)
	}
}

// Contains reports whether item is in the set.
func (s Set[T]) Contains(item T) bool {
	_, ok := s.m[item]
	return ok
}

// Len returns the number of elements in the set.
func (s Set[T]) Len() int {
	return len(s.m)
}

// All returns an iterator over the elements in no particular order.
func (s Set[T]) All() iter.Seq[T] {
	return maps.Keys(s.m)
}

// Slice returns the elements as a slice in no particular order. It returns nil
// for an empty set.
func (s Set[T]) Slice() []T {
	if len(s.m) == 0 {
		return nil
	}
	values := make([]T, 0, len(s.m))
	for value := range s.m {
		values = append(values, value)
	}
	return values
}

// Clone returns an independent copy of the set.
func (s Set[T]) Clone() Set[T] {
	return Set[T]{m: maps.Clone(s.m)}
}

// Equal reports whether s and other contain the same elements.
func (s Set[T]) Equal(other Set[T]) bool {
	return maps.Equal(s.m, other.m)
}

// Union returns a new set containing elements present in either set.
func (s Set[T]) Union(other Set[T]) Set[T] {
	result := s.Clone()
	for item := range other.All() {
		result.Add(item)
	}
	return result
}

// Intersection returns a new set containing elements present in both sets.
func (s Set[T]) Intersection(other Set[T]) Set[T] {
	left, right := s, other
	if right.Len() < left.Len() {
		left, right = right, left
	}
	var result Set[T]
	for item := range left.All() {
		if right.Contains(item) {
			result.Add(item)
		}
	}
	return result
}

// Difference returns a new set containing elements present in s but not other.
func (s Set[T]) Difference(other Set[T]) Set[T] {
	var result Set[T]
	for item := range s.All() {
		if !other.Contains(item) {
			result.Add(item)
		}
	}
	return result
}
