package middleware

import (
	"context"
	"errors"
	"testing"
)

func TestCached_UsesQueryCache(t *testing.T) {
	t.Parallel()

	ctx := NewContext(context.Background())

	calls := 0
	query := func() (int, error) {
		calls++
		return 123, nil
	}

	got1, err := Cached[int](ctx, "k", query)
	if err != nil {
		t.Fatalf("Cached (first): %v", err)
	}
	got2, err := Cached[int](ctx, "k", query)
	if err != nil {
		t.Fatalf("Cached (second): %v", err)
	}

	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
	if got1 != got2 {
		t.Fatalf("expected cached result %v, got %v", got1, got2)
	}
}

func TestCached_DoesNotCacheErrors(t *testing.T) {
	t.Parallel()

	ctx := NewContext(context.Background())

	calls := 0
	sentinel := errors.New("boom")
	query := func() (int, error) {
		calls++
		return 0, sentinel
	}

	_, err := Cached[int](ctx, "k", query)
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}

	_, err = Cached[int](ctx, "k", query)
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}

	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestCached_NoMiddlewareContext_NoCache(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	calls := 0
	query := func() (int, error) {
		calls++
		return 123, nil
	}

	_, _ = Cached[int](ctx, "k", query)
	_, _ = Cached[int](ctx, "k", query)

	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}
