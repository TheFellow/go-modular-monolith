package set_test

import (
	"slices"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/set"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestZeroValue(t *testing.T) {
	t.Parallel()

	var got set.Set[string]
	testutil.Equals(t, got.Len(), 0)
	testutil.IsFalse(t, got.Contains("a"))
	testutil.Nil(t, got.Slice())

	got.Remove("missing")
	got.Add()
	testutil.Nil(t, got.Slice())

	got.Add("a")
	testutil.IsTrue(t, got.Contains("a"))
	got.Remove("a", "missing")
	testutil.Equals(t, got.Len(), 0)
}

func TestNewAndCollect(t *testing.T) {
	t.Parallel()

	fromItems := set.New(1, 2, 2, 3)
	testutil.Equals(t, fromItems.Len(), 3)
	testutil.IsTrue(t, fromItems.Contains(1))
	testutil.IsTrue(t, fromItems.Contains(2))
	testutil.IsTrue(t, fromItems.Contains(3))
	testutil.IsFalse(t, fromItems.Contains(4))

	fromSequence := set.Collect(slices.Values([]string{"a", "b", "a"}))
	testutil.Equals(t, fromSequence.Len(), 2)
	testutil.IsTrue(t, fromSequence.Contains("a"))
	testutil.IsTrue(t, fromSequence.Contains("b"))
}

func TestAllAndSlice(t *testing.T) {
	t.Parallel()

	got := set.New(3, 1, 2)
	testutil.Equals(t, slices.Sorted(got.All()), []int{1, 2, 3})

	fromSlice := got.Slice()
	slices.Sort(fromSlice)
	testutil.Equals(t, fromSlice, []int{1, 2, 3})
}

func TestCloneAndEqual(t *testing.T) {
	t.Parallel()

	original := set.New(1, 2)
	clone := original.Clone()
	testutil.IsTrue(t, original.Equal(clone))

	clone.Add(3)
	clone.Remove(1)
	testutil.IsTrue(t, original.Equal(set.New(2, 1)))
	testutil.IsTrue(t, clone.Equal(set.New(2, 3)))
	testutil.IsFalse(t, original.Equal(clone))
	testutil.IsTrue(t, set.New[int]().Equal(set.Set[int]{}))
}

func TestSetOperations(t *testing.T) {
	t.Parallel()

	left := set.New(1, 2, 3)
	right := set.New(3, 4)

	union := left.Union(right)
	intersection := left.Intersection(right)
	difference := left.Difference(right)

	testutil.IsTrue(t, union.Equal(set.New(1, 2, 3, 4)))
	testutil.IsTrue(t, intersection.Equal(set.New(3)))
	testutil.IsTrue(t, difference.Equal(set.New(1, 2)))

	union.Remove(1)
	intersection.Add(5)
	difference.Add(6)
	testutil.IsTrue(t, left.Equal(set.New(1, 2, 3)))
	testutil.IsTrue(t, right.Equal(set.New(3, 4)))
}

func TestEmptySetOperations(t *testing.T) {
	t.Parallel()

	empty := set.Set[int]{}
	values := set.New(1, 2)

	testutil.IsTrue(t, empty.Union(values).Equal(values))
	testutil.IsTrue(t, values.Union(empty).Equal(values))
	testutil.IsTrue(t, empty.Intersection(values).Equal(empty))
	testutil.IsTrue(t, values.Intersection(empty).Equal(empty))
	testutil.IsTrue(t, empty.Difference(values).Equal(empty))
	testutil.IsTrue(t, values.Difference(empty).Equal(values))
}
