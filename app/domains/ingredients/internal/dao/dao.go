package dao

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

type FileIngredientDAO struct {
	path        string
	ingredients []Ingredient
	loaded      bool
}

const dataPath = "data/ingredients.json"

func New() *FileIngredientDAO {
	return &FileIngredientDAO{path: dataPath}
}

func NewFileIngredientDAO(path string) *FileIngredientDAO {
	return &FileIngredientDAO{path: path}
}

func (d *FileIngredientDAO) ensureLoaded(ctx context.Context) error {
	if d.loaded {
		return nil
	}
	return d.Load(ctx)
}

func (d *FileIngredientDAO) Load(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	b, err := os.ReadFile(d.path)
	if err != nil {
		if os.IsNotExist(err) {
			d.ingredients = []Ingredient{}
			d.loaded = true
			return nil
		}
		return errors.Internalf("read ingredients file: %w", err)
	}
	if len(b) == 0 {
		d.ingredients = []Ingredient{}
		d.loaded = true
		return nil
	}

	var ingredients []Ingredient
	if err := json.Unmarshal(b, &ingredients); err != nil {
		return errors.Internalf("parse ingredients json %q: %w", d.path, err)
	}

	d.ingredients = ingredients
	d.loaded = true
	return nil
}

func (d *FileIngredientDAO) Save(ctx context.Context) error {
	if err := d.ensureLoaded(ctx); err != nil {
		return err
	}

	dir := filepath.Dir(d.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	b, err := json.MarshalIndent(d.ingredients, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	tmp, err := os.CreateTemp(dir, ".ingredients-*.json")
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
