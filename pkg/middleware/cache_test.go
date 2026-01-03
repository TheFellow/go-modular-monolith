package middleware

import (
	"context"
	"errors"
	"testing"

	cedar "github.com/cedar-policy/cedar-go"
)

type testEntity struct {
	uid cedar.EntityUID
	val int
}

func (e testEntity) EntityUID() cedar.EntityUID { return e.uid }

type otherEntity struct {
	uid cedar.EntityUID
}

func (e otherEntity) EntityUID() cedar.EntityUID { return e.uid }

func TestCacheSetGet_ByUID(t *testing.T) {
	t.Parallel()

	ctx := NewContext(context.Background())

	uid := cedar.NewEntityUID(cedar.EntityType("Test::Thing"), cedar.String("a"))
	CacheSet(ctx, testEntity{uid: uid, val: 1})

	got, ok := CacheGet[testEntity](ctx, uid)
	if !ok {
		t.Fatalf("expected cached entity")
	}
	if got.val != 1 {
		t.Fatalf("expected val=1, got %d", got.val)
	}

	if _, ok := CacheGet[otherEntity](ctx, uid); ok {
		t.Fatalf("expected entity type isolation")
	}
}

func TestCachedByUID_CachesSuccess(t *testing.T) {
	t.Parallel()

	ctx := NewContext(context.Background())

	uid := cedar.NewEntityUID(cedar.EntityType("Test::Thing"), cedar.String("a"))

	calls := 0
	fetch := func() (testEntity, error) {
		calls++
		return testEntity{uid: uid, val: 123}, nil
	}

	_, err := CachedByUID(ctx, uid, fetch)
	if err != nil {
		t.Fatalf("CachedByUID (first): %v", err)
	}

	_, err = CachedByUID(ctx, uid, fetch)
	if err != nil {
		t.Fatalf("CachedByUID (second): %v", err)
	}

	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestCachedByUID_DoesNotCacheErrors(t *testing.T) {
	t.Parallel()

	ctx := NewContext(context.Background())

	uid := cedar.NewEntityUID(cedar.EntityType("Test::Thing"), cedar.String("a"))

	calls := 0
	sentinel := errors.New("boom")
	fetch := func() (testEntity, error) {
		calls++
		return testEntity{}, sentinel
	}

	_, err := CachedByUID(ctx, uid, fetch)
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}

	_, err = CachedByUID(ctx, uid, fetch)
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}

	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestCachedByUID_NoMiddlewareContext_NoCache(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	uid := cedar.NewEntityUID(cedar.EntityType("Test::Thing"), cedar.String("a"))

	calls := 0
	fetch := func() (testEntity, error) {
		calls++
		return testEntity{uid: uid, val: 123}, nil
	}

	_, _ = CachedByUID(ctx, uid, fetch)
	_, _ = CachedByUID(ctx, uid, fetch)

	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}
