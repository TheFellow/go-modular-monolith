package optional

type Value[T any] struct {
	valid bool
	v     T
}

func NewSome[T any](v T) Value[T] { return Value[T]{valid: true, v: v} }
func NewNone[T any]() Value[T]    { return Value[T]{} }

func Some[T any](v T) Value[T] { return NewSome(v) }
func None[T any]() Value[T]    { return NewNone[T]() }

func (v Value[T]) IsSome() bool { return v.valid }
func (v Value[T]) IsNone() bool { return !v.valid }

func (v Value[T]) Unwrap() (T, bool) {
	return v.v, v.valid
}

func (v Value[T]) Must() T {
	if !v.valid {
		panic("optional: called Must on None")
	}
	return v.v
}

func (v Value[T]) Or(fallback T) T {
	if v.valid {
		return v.v
	}
	return fallback
}

func (v Value[T]) OrElse(fallback func() T) T {
	if v.valid {
		return v.v
	}
	return fallback()
}

func Map[T, U any](v Value[T], f func(T) U) Value[U] {
	if x, ok := v.Unwrap(); ok {
		return NewSome(f(x))
	}
	return NewNone[U]()
}

func FlatMap[T, U any](v Value[T], f func(T) Value[U]) Value[U] {
	if x, ok := v.Unwrap(); ok {
		return f(x)
	}
	return NewNone[U]()
}
