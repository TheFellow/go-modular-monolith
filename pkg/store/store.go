package store

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
	"github.com/mjl-/bstore"
)

type Store struct {
	db *bstore.DB
}

func Open(ctx context.Context, path string) (*Store, error) {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, errors.Internalf("mkdir db dir: %w", err)
		}
	}

	db, err := bstore.Open(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

// Register adds domain-owned persistence models to this store. Domain module
// bootstrap calls it before the application begins serving operations.
func (s *Store) Register(ctx context.Context, models ...any) {
	if err := s.db.Register(ctx, models...); err != nil {
		panic(err)
	}
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Read(ctx context.Context, fn func(*bstore.Tx) error) error {
	start := time.Now()
	err := s.db.Read(ctx, fn)
	telemetry.FromContext(ctx).Histogram(telemetry.MetricStoreReadDuration).ObserveDuration(start)
	return err
}

func (s *Store) Write(ctx context.Context, fn func(*bstore.Tx) error) error {
	start := time.Now()
	err := s.db.Write(ctx, fn)
	telemetry.FromContext(ctx).Histogram(telemetry.MetricStoreWriteDuration).ObserveDuration(start)
	return err
}
