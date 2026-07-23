package testutil

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
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

// NotEquals fails the test when got and want are equal.
func NotEquals[T any](t testing.TB, got, want T, opts ...cmp.Option) {
	t.Helper()
	all := append([]cmp.Option{cmpopts.EquateErrors(), allowOptionalUnexported()}, opts...)
	diff := cmp.Diff(want, got, all...)
	ErrorIf(t, diff == "", "unexpected equality: both values are equal: %v", got)
}

// IsTrue fails the test when got is false.
func IsTrue(t testing.TB, got bool) {
	t.Helper()
	ErrorIf(t, !got, "expected true, got false")
}

// IsFalse fails the test when got is true.
func IsFalse(t testing.TB, got bool) {
	t.Helper()
	ErrorIf(t, got, "expected false, got true")
}

// EquateAmounts compares measurement amounts through their public unit and
// value, allowing for floating-point conversion error.
func EquateAmounts(tolerance float64) cmp.Option {
	return cmp.Comparer(func(x, y measurement.Amount) bool {
		if x == nil || y == nil {
			return x == nil && y == nil
		}
		return x.Unit() == y.Unit() && math.Abs(x.Value()-y.Value()) <= tolerance
	})
}

// Cast asserts that value has type T and returns it.
func Cast[T any](t testing.TB, value any) T {
	t.Helper()
	result, ok := value.(T)
	ErrorIf(t, !ok, "wrong type %T, want %s", value, reflect.TypeFor[T]())
	return result
}

// Nil fails the test when value is not nil, including typed nil values.
func Nil(t testing.TB, value any) {
	t.Helper()
	ErrorIf(t, !isNil(value), "expected nil, got %v", value)
}

// NotNil fails the test when value is nil, including typed nil values.
func NotNil(t testing.TB, value any) {
	t.Helper()
	ErrorIf(t, isNil(value), "expected non-nil value, got nil")
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

// ErrorContains fails the test when err is nil or its message does not contain
// substr.
func ErrorContains(t testing.TB, err error, substr string) {
	t.Helper()
	NotNil(t, err)
	ErrorIf(t, !strings.Contains(err.Error(), substr), "expected error %q to contain %q", err.Error(), substr)
}

// StringContains fails the test when value does not contain substr.
func StringContains(t testing.TB, value, substr string) {
	t.Helper()
	ErrorIf(t, !strings.Contains(value, substr), "expected string %q to contain %q", value, substr)
}

// ExpectPanic fails the test unless f panics with expected.
func ExpectPanic(t testing.TB, expected string, f func()) {
	t.Helper()
	defer func() {
		got := recover()
		NotNil(t, got)
		Equals(t, got, any(expected))
	}()
	f()
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

func isNil(value any) bool {
	if value == nil {
		return true
	}

	rv := reflect.ValueOf(value)
	//nolint:exhaustive // Only nillable kinds are relevant.
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}
