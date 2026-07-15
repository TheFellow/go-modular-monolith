// Package optional provides a minimal generic value that distinguishes an
// absent value from a present value containing T's zero value.
//
// The zero value of Value[T] is None. Construct values with Some and None,
// inspect their state with IsSome and IsNone, and retrieve values with Unwrap.
package optional
