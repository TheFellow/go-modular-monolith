package testutil

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Equals(t testing.TB, got, want any, opts ...cmp.Option) {
	diff := cmp.Diff(want, got, opts...)
	ErrorIf(t, diff != "", "mismatch (-want +got):\n%s", diff)
}

func ErrorIf(t testing.TB, isErr bool, msg string, args ...any) {
	if isErr {
		t.Fatalf(msg, args...)
	}
}

func Ok(t testing.TB, err error) {
	ErrorIf(t, err != nil, "unexpected error: %v", err)
}
