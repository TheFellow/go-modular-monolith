package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type FileDrinkDAO struct {
	path string
}

func NewFileDrinkDAO(path string) *FileDrinkDAO {
	return &FileDrinkDAO{path: path}
}

func (d *FileDrinkDAO) Load(ctx context.Context) ([]Drink, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b, err := os.ReadFile(d.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Drink{}, nil
		}
		return nil, err
	}
	if len(b) == 0 {
		return []Drink{}, nil
	}

	var drinks []Drink
	if err := json.Unmarshal(b, &drinks); err != nil {
		return nil, fmt.Errorf("parse drinks json %q: %w", d.path, err)
	}
	return drinks, nil
}

func (d *FileDrinkDAO) Save(ctx context.Context, drinks []Drink) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	dir := filepath.Dir(d.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	b, err := json.MarshalIndent(drinks, "", "  ")
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
