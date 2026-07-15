package errors

import stderrors "errors"

// Standard-library forwarding helpers let callers use one errors import.
func As(err error, target any) bool { return stderrors.As(err, target) }

func AsType[E error](err error) (E, bool) { return stderrors.AsType[E](err) }

func Is(err, target error) bool { return stderrors.Is(err, target) }

func Join(errs ...error) error { return stderrors.Join(errs...) }

func New(text string) error { return stderrors.New(text) }

func Unwrap(err error) error { return stderrors.Unwrap(err) }
