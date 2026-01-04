package optional

//sumtype:decl
type Value[T any] interface {
	IsSome() bool
	IsNone() bool
	Unwrap() (T, bool)
	Must() T
	Or(fallback T) T
	OrElse(fallback func() T) T
	sealed()
}

type Some[T any] struct {
	Val T
}

func NewSome[T any](v T) Some[T] { return Some[T]{Val: v} }

func (Some[T]) IsSome() bool          { return true }
func (Some[T]) IsNone() bool          { return false }
func (s Some[T]) Unwrap() (T, bool)   { return s.Val, true }
func (s Some[T]) Must() T             { return s.Val }
func (s Some[T]) Or(_ T) T            { return s.Val }
func (s Some[T]) OrElse(_ func() T) T { return s.Val }
func (Some[T]) sealed()               {}

type None[T any] struct{}

func NewNone[T any]() None[T] { return None[T]{} }

func (None[T]) IsSome() bool          { return false }
func (None[T]) IsNone() bool          { return true }
func (None[T]) Unwrap() (T, bool)     { var zero T; return zero, false }
func (None[T]) Must() T               { panic("optional: called Must on None") }
func (n None[T]) Or(fallback T) T     { return fallback }
func (n None[T]) OrElse(f func() T) T { return f() }
func (None[T]) sealed()               {}

func Map[T, U any](v Value[T], f func(T) U) Value[U] {
	if v == nil {
		return NewNone[U]()
	}
	switch x := v.(type) {
	case Some[T]:
		return NewSome(f(x.Val))
	case None[T]:
		return NewNone[U]()
	}
	return NewNone[U]()
}

func FlatMap[T, U any](v Value[T], f func(T) Value[U]) Value[U] {
	if v == nil {
		return NewNone[U]()
	}
	switch x := v.(type) {
	case Some[T]:
		return f(x.Val)
	case None[T]:
		return NewNone[U]()
	}
	return NewNone[U]()
}
