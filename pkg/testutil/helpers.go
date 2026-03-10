package testutil

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Equals(t testing.TB, got, want any, opts ...cmp.Option) {
	t.Helper()
	diff := cmp.Diff(want, got, opts...)
	ErrorIf(t, diff != "", "mismatch (-want +got):\n%s", diff)
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
