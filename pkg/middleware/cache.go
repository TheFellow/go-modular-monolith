package middleware

import "context"

type QueryKey string

type QueryCache struct {
	cache map[QueryKey]any
}

func newQueryCache() *QueryCache {
	return &QueryCache{cache: make(map[QueryKey]any)}
}

func (qc *QueryCache) Get(key QueryKey) (any, bool) {
	v, ok := qc.cache[key]
	return v, ok
}

func (qc *QueryCache) Set(key QueryKey, value any) {
	qc.cache[key] = value
}

// Cached memoizes query results for the lifetime of a single execution context.
//
// When ctx is a *middleware.Context, results are stored in that context's
// per-execution cache; otherwise, the query runs without caching.
func Cached[T any](ctx context.Context, key string, query func() (T, error)) (T, error) {
	if ctx == nil {
		return query()
	}

	mctx, ok := ctx.(*Context)
	if !ok || mctx.queryCache == nil {
		return query()
	}

	qkey := QueryKey(key)
	if cached, ok := mctx.queryCache.Get(qkey); ok {
		if typed, ok := cached.(T); ok {
			return typed, nil
		}
	}

	result, err := query()
	if err == nil {
		mctx.queryCache.Set(qkey, result)
	}
	return result, err
}
