package dao

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

type FileStockDAO struct {
	path   string
	stock  []Stock
	loaded bool
}

const dataPath = "data/stock.json"

func New() *FileStockDAO {
	return &FileStockDAO{path: dataPath}
}

func NewFileStockDAO(path string) *FileStockDAO {
	return &FileStockDAO{path: path}
}

func (d *FileStockDAO) ensureLoaded(ctx context.Context) error {
	if d.loaded {
		return nil
	}
	return d.Load(ctx)
}

func (d *FileStockDAO) Load(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	b, err := os.ReadFile(d.path)
	if err != nil {
		if os.IsNotExist(err) {
			d.stock = []Stock{}
			d.loaded = true
			return nil
		}
		return errors.Internalf("read stock file: %w", err)
	}
	if len(b) == 0 {
		d.stock = []Stock{}
		d.loaded = true
		return nil
	}

	var stock []Stock
	if err := json.Unmarshal(b, &stock); err != nil {
		return errors.Internalf("parse stock json %q: %w", d.path, err)
	}

	d.stock = stock
	d.loaded = true
	return nil
}

func (d *FileStockDAO) Save(ctx context.Context) error {
	if err := d.ensureLoaded(ctx); err != nil {
		return err
	}

	dir := filepath.Dir(d.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	b, err := json.MarshalIndent(d.stock, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	tmp, err := os.CreateTemp(dir, ".stock-*.json")
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
