package paging_test

import (
	"strconv"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestCountTraversesPages(t *testing.T) {
	t.Parallel()

	count, err := paging.Count(func(cursor paging.Cursor) (paging.Page[int], error) {
		page := 0
		if cursor != "" {
			var err error
			page, err = strconv.Atoi(string(cursor))
			if err != nil {
				return paging.Page[int]{}, err
			}
		}
		if page == 2 {
			return paging.Page[int]{Items: []int{5}}, nil
		}
		return paging.Page[int]{Items: []int{page*2 + 1, page*2 + 2}, Next: paging.Cursor(strconv.Itoa(page + 1))}, nil
	})

	testutil.Ok(t, err)
	testutil.Equals(t, count, 5)
}
