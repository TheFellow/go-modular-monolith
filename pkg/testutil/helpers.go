package testutil

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var optionalValuePkgPath = reflect.TypeFor[optional.Value[int]]().PkgPath()

// allowOptionalUnexported lets cmp compare every instantiation of
// optional.Value[T] without exposing the type's invariant-bearing fields.
func allowOptionalUnexported() cmp.Option {
	return cmp.Exporter(func(t reflect.Type) bool {
		return t.PkgPath() == optionalValuePkgPath && strings.HasPrefix(t.Name(), "Value[")
	})
}

// Equals fails the test when got and want differ. Application errors are
// compared through errors.Is, and optional values may retain private fields.
func Equals[T any](t testing.TB, got, want T, opts ...cmp.Option) {
	t.Helper()
	all := append([]cmp.Option{cmpopts.EquateErrors(), allowOptionalUnexported()}, opts...)
	diff := cmp.Diff(want, got, all...)
	ErrorIf(t, diff != "", "mismatch (-want +got):\n%s", diff)
}

// ErrorAs fails the test when err cannot be assigned to target through its
// error chain.
func ErrorAs(t testing.TB, err error, target any) {
	t.Helper()
	ErrorIf(t, !errors.As(err, target), "got %v, want %v", err, target)
}

// ErrorIs fails the test when err does not match target through its error
// chain.
func ErrorIs(t testing.TB, err, target error) {
	t.Helper()
	ErrorIf(t, !errors.Is(err, target), "expected error %v to match target %v", err, target)
}

func ErrorIf(t testing.TB, isErr bool, msg string, args ...any) {
	t.Helper()
	if isErr {
		t.Fatalf(msg, args...)
	}
}

func StringNonEmpty(t testing.TB, value string, msg string, args ...any) {
	t.Helper()
	if strings.TrimSpace(value) == "" {
		t.Fatalf(msg, args...)
	}
}

func Ok(t testing.TB, err error) {
	t.Helper()
	ErrorIf(t, err != nil, "unexpected error: %v", err)
}
