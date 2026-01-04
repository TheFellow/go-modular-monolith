package dao

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

type FileDrinkDAO struct {
	path   string
	drinks []Drink
	loaded bool
}

const dataPath = "data/drinks.json"

func New() *FileDrinkDAO {
	return &FileDrinkDAO{path: dataPath}
}

func NewFileDrinkDAO(path string) *FileDrinkDAO {
	return &FileDrinkDAO{path: path}
}

func (d *FileDrinkDAO) ensureLoaded(ctx context.Context) error {
	if d.loaded {
		return nil
	}
	return d.Load(ctx)
}

func (d *FileDrinkDAO) Load(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	b, err := os.ReadFile(d.path)
	if err != nil {
		if os.IsNotExist(err) {
			d.drinks = []Drink{}
			d.loaded = true
			return nil
		}
		return errors.Internalf("read drinks file: %w", err)
	}
	if len(b) == 0 {
		d.drinks = []Drink{}
		d.loaded = true
		return nil
	}

	var drinks []Drink
	if err := json.Unmarshal(b, &drinks); err != nil {
		return errors.Internalf("parse drinks json %q: %w", d.path, err)
	}

	d.drinks = drinks
	d.loaded = true
	return nil
}

func (d *FileDrinkDAO) Save(ctx context.Context) error {
	if err := d.ensureLoaded(ctx); err != nil {
		return err
	}

	dir := filepath.Dir(d.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	b, err := json.MarshalIndent(d.drinks, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	tmp, err := os.CreateTemp(dir, ".drinks-*.json")
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
