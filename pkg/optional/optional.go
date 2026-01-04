package optional

type Value[T any] struct {
	valid bool
	v     T
}

func Some[T any](v T) Value[T] { return Value[T]{valid: true, v: v} }
func None[T any]() Value[T]    { return Value[T]{} }

func (v Value[T]) IsSome() bool { return v.valid }
func (v Value[T]) IsNone() bool { return !v.valid }

func (v Value[T]) Unwrap() (T, bool) {
	return v.v, v.valid
}

func (v Value[T]) Must() T {
	if !v.valid {
		panic("option is None")
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
		return Some(f(x))
	}
	return None[U]()
}

func FlatMap[T, U any](v Value[T], f func(T) Value[U]) Value[U] {
	if x, ok := v.Unwrap(); ok {
		return f(x)
	}
	return None[U]()
}
