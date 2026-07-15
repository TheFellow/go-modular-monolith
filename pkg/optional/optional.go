package optional

type Value[T any] struct {
	value T
	valid bool
}

// Some constructs a present optional value, including when value is T's zero
// value.
func Some[T any](value T) Value[T] {
	return Value[T]{value: value, valid: true}
}

// None constructs an absent optional value. The zero value of Value[T] is
// also None.
func None[T any]() Value[T] { return Value[T]{} }

// IsSome reports whether the optional contains a value.
func (v Value[T]) IsSome() bool { return v.valid }

// IsNone reports whether the optional is absent.
func (v Value[T]) IsNone() bool { return !v.valid }

// Unwrap returns the wrapped value and true when present, or T's zero value
// and false when absent.
func (v Value[T]) Unwrap() (T, bool) {
	return v.value, v.valid
}
