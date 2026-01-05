package store

import (
	"context"
	"os"
	"path/filepath"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/mjl-/bstore"
)

type Store struct {
	db *bstore.DB
}

func Open(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, errors.Internalf("mkdir db dir: %w", err)
		}
	}

	db, err := bstore.Open(context.Background(), path, nil)
	if err != nil {
		return nil, err
	}

	types := registeredTypes()
	if len(types) == 0 {
		_ = db.Close()
		return nil, errors.Internalf("no bstore types registered (expected internal DAO packages to call store.RegisterTypes in init)")
	}
	if err := db.Register(context.Background(), types...); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Read(ctx context.Context, fn func(*bstore.Tx) error) error {
	if s == nil || s.db == nil {
		return errors.Internalf("store not initialized")
	}
	return s.db.Read(ctx, fn)
}

func (s *Store) Write(ctx context.Context, fn func(*bstore.Tx) error) error {
	if s == nil || s.db == nil {
		return errors.Internalf("store not initialized")
	}
	return s.db.Write(ctx, fn)
}

func (s *Store) DB() *bstore.DB {
	if s == nil {
		return nil
	}
	return s.db
}
