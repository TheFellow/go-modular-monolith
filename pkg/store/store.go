package store

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/mjl-/bstore"
)

var (
	mu sync.Mutex
	DB *bstore.DB
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

func Open(path string) error {
	mu.Lock()
	defer mu.Unlock()

	if DB != nil {
		return errors.Internalf("store already open")
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return errors.Internalf("mkdir db dir: %w", err)
		}
	}

	db, err := bstore.Open(context.Background(), path, nil)
	if err != nil {
		return err
	}
	types := registeredTypes()
	if len(types) == 0 {
		_ = db.Close()
		return errors.Internalf("no bstore types registered (expected internal DAO packages to call store.RegisterTypes in init)")
	}
	if err := db.Register(context.Background(), types...); err != nil {
		_ = db.Close()
		return err
	}
	DB = db
	return nil
}

func Register(ctx context.Context, types ...any) error {
	mu.Lock()
	db := DB
	mu.Unlock()

	if db == nil {
		return errors.Internalf("store not initialized")
	}
	if len(types) == 0 {
		return errors.Internalf("missing bstore types")
	}
	return db.Register(ctx, types...)
}

func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if DB == nil {
		return nil
	}
	err := DB.Close()
	DB = nil
	return err
}
