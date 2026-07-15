package optional_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func TestSome(t *testing.T) {
	t.Parallel()

	value := optional.Some("hi")
	if !value.IsSome() || value.IsNone() {
		t.Fatal("Some value reported the wrong state")
	}

	got, ok := value.Unwrap()
	if !ok || got != "hi" {
		t.Fatalf("Unwrap() = %q, %v; want hi, true", got, ok)
	}
}

func TestNone(t *testing.T) {
	t.Parallel()

	value := optional.None[string]()
	if value.IsSome() || !value.IsNone() {
		t.Fatal("None value reported the wrong state")
	}

	got, ok := value.Unwrap()
	if ok || got != "" {
		t.Fatalf("Unwrap() = %q, %v; want empty string, false", got, ok)
	}
}

func TestSomeZeroValue(t *testing.T) {
	t.Parallel()

	got, ok := optional.Some(0).Unwrap()
	if !ok || got != 0 {
		t.Fatalf("Unwrap() = %d, %v; want 0, true", got, ok)
	}
}

func TestZeroValueIsNone(t *testing.T) {
	t.Parallel()

	var value optional.Value[int]
	if value.IsSome() || !value.IsNone() {
		t.Fatal("zero value reported the wrong state")
	}
	got, ok := value.Unwrap()
	if ok || got != 0 {
		t.Fatalf("Unwrap() = %d, %v; want 0, false", got, ok)
	}
}
