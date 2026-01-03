package middleware

import (
	"context"

	cedar "github.com/cedar-policy/cedar-go"
)

type EntityCache struct {
	entities map[string]any
}

func newEntityCache() *EntityCache {
	return &EntityCache{entities: make(map[string]any)}
}

func (c *EntityCache) Get(uid cedar.EntityUID) (any, bool) {
	if c == nil {
		return nil, false
	}
	v, ok := c.entities[entityKey(uid)]
	return v, ok
}

func (c *EntityCache) Set(entity CedarEntity) {
	if c == nil {
		return
	}
	c.entities[entityKey(entity.EntityUID())] = entity
}

func cacheFromContext(ctx context.Context) *EntityCache {
	if ctx == nil {
		return nil
	}
	mctx, ok := ctx.(*Context)
	if !ok || mctx.entityCache == nil {
		return nil
	}
	return mctx.entityCache
}

type CedarEntity interface {
	EntityUID() cedar.EntityUID
}

func entityKey(uid cedar.EntityUID) string {
	return string(uid.Type) + "\x00" + string(uid.ID)
}

func CacheGet[T any](ctx context.Context, uid cedar.EntityUID) (T, bool) {
	var zero T

	cache := cacheFromContext(ctx)
	if cache == nil {
		return zero, false
	}

	v, ok := cache.Get(uid)
	if !ok {
		return zero, false
	}

	typed, ok := v.(T)
	if !ok {
		return zero, false
	}
	return typed, true
}

func CacheSet[T CedarEntity](ctx context.Context, entity T) {
	cache := cacheFromContext(ctx)
	if cache == nil {
		return
	}
	cache.Set(entity)
}

func CacheSetAll[T CedarEntity](ctx context.Context, entities []T) {
	for _, entity := range entities {
		CacheSet(ctx, entity)
	}
}

func CachedByUID[T CedarEntity](ctx context.Context, uid cedar.EntityUID, fetch func() (T, error)) (T, error) {
	if cached, ok := CacheGet[T](ctx, uid); ok {
		return cached, nil
	}

	entity, err := fetch()
	if err != nil {
		return entity, err
	}

	CacheSet(ctx, entity)
	return entity, nil
}
