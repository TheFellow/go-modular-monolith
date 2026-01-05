package testutil

import (
	"path/filepath"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func OpenStore(t testing.TB) *store.Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "mixology.test.db")
	s, err := store.Open(path)
	Ok(t, err)
	t.Cleanup(func() {
		_ = s.Close()
	})
	return s
}
