package dao

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

type FileOrderDAO struct {
	path   string
	orders []Order
	loaded bool
}

const dataPath = "data/orders.json"

func New() *FileOrderDAO {
	return &FileOrderDAO{path: dataPath}
}

func NewFileOrderDAO(path string) *FileOrderDAO {
	return &FileOrderDAO{path: path}
}

func (d *FileOrderDAO) ensureLoaded(ctx context.Context) error {
	if d.loaded {
		return nil
	}
	return d.Load(ctx)
}

func (d *FileOrderDAO) Load(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	b, err := os.ReadFile(d.path)
	if err != nil {
		if os.IsNotExist(err) {
			d.orders = []Order{}
			d.loaded = true
			return nil
		}
		return errors.Internalf("read orders file: %w", err)
	}
	if len(b) == 0 {
		d.orders = []Order{}
		d.loaded = true
		return nil
	}

	var orders []Order
	if err := json.Unmarshal(b, &orders); err != nil {
		return errors.Internalf("parse orders json %q: %w", d.path, err)
	}

	d.orders = orders
	d.loaded = true
	return nil
}

func (d *FileOrderDAO) Save(ctx context.Context) error {
	if err := d.ensureLoaded(ctx); err != nil {
		return err
	}

	dir := filepath.Dir(d.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	b, err := json.MarshalIndent(d.orders, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	tmp, err := os.CreateTemp(dir, ".orders-*.json")
	if err != nil {
		return err
	}

	tmpName := tmp.Name()
	_, writeErr := tmp.Write(b)
	closeErr := tmp.Close()
	if writeErr != nil {
		_ = os.Remove(tmpName)
		return writeErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpName)
		return closeErr
	}

	if err := os.Chmod(tmpName, 0o644); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, d.path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}

	return nil
}
