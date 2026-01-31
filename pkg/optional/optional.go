package optional

type Value[T any] struct {
	Valid bool
	Val   T
}

func NewSome[T any](v T) Value[T] { return Value[T]{Valid: true, Val: v} }
func NewNone[T any]() Value[T]    { return Value[T]{} }

func Some[T any](v T) Value[T] { return NewSome(v) }
func None[T any]() Value[T]    { return NewNone[T]() }

func (v Value[T]) IsSome() bool { return v.Valid }
func (v Value[T]) IsNone() bool { return !v.Valid }

func (v Value[T]) Unwrap() (T, bool) {
	return v.Val, v.Valid
}

func (v Value[T]) Must() T {
	if !v.Valid {
		panic("optional: called Must on None")
	}
	return v.Val
}

func (v Value[T]) Or(fallback T) T {
	if v.Valid {
		return v.Val
	}
	return fallback
}

func (v Value[T]) OrElse(fallback func() T) T {
	if v.Valid {
		return v.Val
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
