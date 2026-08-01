// Package assert provides dependency-free test assertions for packages that
// cannot import the root testutil package without creating an import cycle.
package assert

import "testing"

// ErrorIf fails the test when isErr is true.
func ErrorIf(t testing.TB, isErr bool, msg string, args ...any) {
	t.Helper()
	if isErr {
		t.Fatalf(msg, args...)
	}
}
