package store

import (
	"context"

	"github.com/mjl-/bstore"
)

type txKey struct{}

func WithTx(ctx context.Context, tx *bstore.Tx) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, txKey{}, tx)
}

func TxFromContext(ctx context.Context) (*bstore.Tx, bool) {
	if ctx == nil {
		return nil, false
	}
	tx, ok := ctx.Value(txKey{}).(*bstore.Tx)
	return tx, ok
}

type storeKey struct{}

func WithStore(ctx context.Context, s *Store) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, storeKey{}, s)
}

func FromContext(ctx context.Context) (*Store, bool) {
	if ctx == nil {
		return nil, false
	}
	s, ok := ctx.Value(storeKey{}).(*Store)
	return s, ok
}
