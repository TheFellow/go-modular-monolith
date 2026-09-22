package optional_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestSome(t *testing.T) {
	t.Parallel()

	value := optional.Some("hi")
	testutil.IsTrue(t, value.IsSome())
	testutil.IsFalse(t, value.IsNone())

	got, ok := value.Unwrap()
	testutil.IsTrue(t, ok)
	testutil.Equals(t, got, "hi")
}

func TestNone(t *testing.T) {
	t.Parallel()

	value := optional.None[string]()
	testutil.IsFalse(t, value.IsSome())
	testutil.IsTrue(t, value.IsNone())

	got, ok := value.Unwrap()
	testutil.IsFalse(t, ok)
	testutil.Equals(t, got, "")
}

func TestSomeZeroValue(t *testing.T) {
	t.Parallel()

	got, ok := optional.Some(0).Unwrap()
	testutil.IsTrue(t, ok)
	testutil.Equals(t, got, 0)
}

func TestZeroValueIsNone(t *testing.T) {
	t.Parallel()

	var value optional.Value[int]
	testutil.IsFalse(t, value.IsSome())
	testutil.IsTrue(t, value.IsNone())
	got, ok := value.Unwrap()
	testutil.IsFalse(t, ok)
	testutil.Equals(t, got, 0)
}

func TestSet(t *testing.T) {
	t.Parallel()
	var value optional.Value[int]
	value.Set(42)
	got, present := value.Unwrap()
	testutil.IsTrue(t, present)
	testutil.Equals(t, got, 42)
	value.Set(0)
	got, present = value.Unwrap()
	testutil.IsTrue(t, present)
	testutil.Equals(t, got, 0)
}
