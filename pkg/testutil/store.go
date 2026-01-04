package testutil

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func OpenStore(t testing.TB) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "mixology.test.db")
	Ok(t, store.Open(path))
	Ok(t, store.Register(context.Background(), app.StoreTypes()...))
	t.Cleanup(func() {
		_ = store.Close()
	})
}
