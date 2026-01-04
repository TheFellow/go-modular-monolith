package dao

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

type FileMenuDAO struct {
	path   string
	menus  []Menu
	loaded bool
}

const dataPath = "data/menus.json"

func New() *FileMenuDAO {
	return &FileMenuDAO{path: dataPath}
}

func NewFileMenuDAO(path string) *FileMenuDAO {
	return &FileMenuDAO{path: path}
}

func (d *FileMenuDAO) ensureLoaded(ctx context.Context) error {
	if d.loaded {
		return nil
	}
	return d.Load(ctx)
}

func (d *FileMenuDAO) Load(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	b, err := os.ReadFile(d.path)
	if err != nil {
		if os.IsNotExist(err) {
			d.menus = []Menu{}
			d.loaded = true
			return nil
		}
		return errors.Internalf("read menus file: %w", err)
	}
	if len(b) == 0 {
		d.menus = []Menu{}
		d.loaded = true
		return nil
	}

	var menus []Menu
	if err := json.Unmarshal(b, &menus); err != nil {
		return errors.Internalf("parse menus json %q: %w", d.path, err)
	}

	d.menus = menus
	d.loaded = true
	return nil
}

func (d *FileMenuDAO) Save(ctx context.Context) error {
	if err := d.ensureLoaded(ctx); err != nil {
		return err
	}

	dir := filepath.Dir(d.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	b, err := json.MarshalIndent(d.menus, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	tmp, err := os.CreateTemp(dir, ".menus-*.json")
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

type Menu struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Items       []MenuItem `json:"items,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

type MenuItem struct {
	DrinkID      string `json:"drink_id"`
	DisplayName  string `json:"display_name,omitempty"`
	Price        *Price `json:"price,omitempty"`
	Featured     bool   `json:"featured,omitempty"`
	Availability string `json:"availability"`
	SortOrder    int    `json:"sort_order,omitempty"`
}

type Price struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}
